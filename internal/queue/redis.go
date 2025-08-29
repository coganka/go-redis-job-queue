package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"redis-job-queue/internal/config"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type RedisQueue struct {
	client        *redis.Client
	stream        string
	consumerGroup string
}

func (q *RedisQueue) Client() *redis.Client {
	return q.client
}

func NewRedisClient(cfg config.Config) *RedisQueue {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   cfg.RedisDB,
	})

	// test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connect failed: %v", err)
	}

	// ensure consumer group exists
	_ = rdb.XGroupCreateMkStream(ctx, cfg.Stream, cfg.ConsumerGroup, "$").Err()

	return &RedisQueue{
		client:        rdb,
		stream:        cfg.Stream,
		consumerGroup: cfg.ConsumerGroup,
	}
}

func (q *RedisQueue) Enqueue(jobType string, payload json.RawMessage, scheduledAt int64) (string, error) {
	jobID := uuid.NewString()
	env := JobEnvelope{
		ID:          jobID,
		Type:        jobType,
		Payload:     payload,
		Attempt:     0,
		MaxAttempts: 5,
		TimeoutMS:   30000,
		CreatedAt:   time.Now().Unix(),
		ScheduledAt: scheduledAt,
	}

	jobJSON, _ := json.Marshal(env)

	if scheduledAt > time.Now().Unix() {
		// future job -> put in scheduled ZSET
		_, err := q.client.ZAdd(ctx, "jobs:scheduled", redis.Z{
			Score:  float64(scheduledAt),
			Member: jobJSON,
		}).Result()
		return jobID, err
	}

	// immediate job -> normal stream
	_, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{"job": string(jobJSON)},
	}).Result()

	if err != nil {
		return "", err
	}
	return jobID, nil
}
