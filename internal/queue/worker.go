package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"redis-job-queue/internal/store"

	"github.com/redis/go-redis/v9"
)

type Worker struct {
	q       *RedisQueue
	name    string
	workers int
	store   *store.Store
}

func NewWorker(q *RedisQueue, name string, workers int, s *store.Store) *Worker {
	return &Worker{q: q, name: name, workers: workers, store: s}
}

// start launches worker goroutines
func (w *Worker) Start() {
	for i := 0; i < w.workers; i++ {
		go w.run(i)
	}
}

func (w *Worker) run(id int) {
	log.Printf("[worker-%d] started", id)
	for {
		entries, err := w.q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.q.consumerGroup,
			Consumer: w.name,
			Streams:  []string{w.q.stream, ">"},
			Block:    5 * time.Second,
			Count:    1,
		}).Result()

		if err != nil && err != redis.Nil {
			log.Printf("[worker-%d] XREADGROUP error: %v", id, err)
			continue
		}
		if len(entries) == 0 {
			continue // timeout, loop again
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				raw, ok := msg.Values["job"].(string)
				if !ok {
					log.Printf("[worker-%d] bad message format", id)
					continue
				}

				var job JobEnvelope
				if err := json.Unmarshal([]byte(raw), &job); err != nil {
					log.Printf("[worker-%d] bad JSON: %v", id, err)
					continue
				}

				// mark as processing
				_ = w.store.SetStatus(job.ID, "processing", map[string]interface{}{
					"started_at": time.Now().Unix(),
					"attempts":   job.Attempt,
				})

				// handle
				err := w.handleJob(job)
				if err != nil {
					job.Attempt++ // bump attempt count

					if job.Attempt < job.MaxAttempts {
						// schedule retry
						delay := RetryDelay(1*time.Second, job.Attempt)
						retryAt := time.Now().Add(delay).Unix()

						jobJSON, _ := json.Marshal(job)
						if zerr := w.q.client.ZAdd(ctx, "jobs:retry", redis.Z{
							Score:  float64(retryAt),
							Member: jobJSON,
						}).Err(); zerr != nil {
							log.Printf("[worker-%d] failed to enqueue retry: %v", id, zerr)
						} else {
							log.Printf("[worker-%d] job %s scheduled for retry in %v", id, job.ID, delay)

							_ = w.store.SetStatus(job.ID, "retrying", map[string]interface{}{
								"last_error": err.Error(),
								"attempts":   job.Attempt,
							})

						}

					} else {
						// final failure -> DLQ
						log.Printf("[worker-%d] job %s reached max attempts, moving to DLQ", id, job.ID)

						_ = w.store.SetStatus(job.ID, "failed", map[string]interface{}{
							"last_error":  err.Error(),
							"finished_at": time.Now().Unix(),
							"attempts":    job.Attempt,
						})

						jobJSON, _ := json.Marshal(job)
						if err := w.q.client.XAdd(ctx, &redis.XAddArgs{
							Stream: "jobs:dlq",
							Values: map[string]interface{}{"job": string(jobJSON)},
						}).Err(); err != nil {
							log.Printf("[worker-%d] DLQ push failed: %v", id, err)
						}
					}

				} else {
					_ = w.store.SetStatus(job.ID, "succeeded", map[string]interface{}{
						"finished_at": time.Now().Unix(),
						"attempts":    job.Attempt,
					})
				}

				// ack
				if err := w.q.client.XAck(ctx, w.q.stream, w.q.consumerGroup, msg.ID).Err(); err != nil {
					log.Printf("[worker-%d] XACK failed: %v", id, err)
				}
			}
		}
	}
}

// handleJob dispatches by job type
func (w *Worker) handleJob(job JobEnvelope) error {
	log.Printf("[worker] processing job %s type=%s payload=%s",
		job.ID, job.Type, string(job.Payload))

	if job.Type == "echo.process" {
		// simulate work
		time.Sleep(1 * time.Second)
		log.Printf("[worker] echo done: %s", string(job.Payload))
		return nil
	}

	// anything else = fail
	return fmt.Errorf("unknown job type: %s", job.Type)
}
