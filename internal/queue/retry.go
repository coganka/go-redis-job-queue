package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"redis-job-queue/internal/store"

	"math/rand"

	"github.com/redis/go-redis/v9"
)

func RetryDelay(base time.Duration, attempt int) time.Duration {
	delay := base * (1 << attempt)
	jitter := time.Duration(rand.Int63n(int64(base)))
	return delay + jitter
}

func StartRetryManager(rdb *redis.Client, stream string, s *store.Store) {
	go func() {
		ctx := context.Background()

		for {
			// look for jobs ready to retry
			now := time.Now().Unix()
			jobs, err := rdb.ZRangeByScore(ctx, "jobs:retry", &redis.ZRangeBy{
				Min:    "-inf",
				Max:    fmt.Sprint(now),
				Offset: 0,
				Count:  10,
			}).Result()

			if err != nil && err != redis.Nil {
				log.Printf("[retry-manager] ZRANGEBYSCORE error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(jobs) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}

			for _, raw := range jobs {
				var job JobEnvelope
				if err := json.Unmarshal([]byte(raw), &job); err != nil {
					log.Printf("[retry-manager] bad JSON: %v", err)
					_, _ = rdb.ZRem(ctx, "jobs:retry", raw).Result()
					continue
				}

				// re-enqueue job
				_, err := rdb.XAdd(ctx, &redis.XAddArgs{
					Stream: stream,
					Values: map[string]interface{}{"job": raw},
				}).Result()
				if err != nil {
					log.Printf("[retry-manager] XADD failed: %v", err)
					continue
				}

				// remove from retry ZSET
				_, _ = rdb.ZRem(ctx, "jobs:retry", raw).Result()

				// Important! only set queued if job is still < max_attempts
				if job.Attempt < job.MaxAttempts {
					_ = s.SetStatus(job.ID, "queued")
				}

				log.Printf("[retry-manager] re-enqueued job %s (attempt=%d)", job.ID, job.Attempt)

			}
		}
	}()
}
