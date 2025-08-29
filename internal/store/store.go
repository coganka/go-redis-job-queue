package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Store struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func (s *Store) SetStatus(jobID, status string, fields ...map[string]interface{}) error {
	key := "job:" + jobID

	data := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().Unix(),
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			data[k] = v
		}
	}

	return s.rdb.HSet(ctx, key, data).Err()
}

func (s *Store) GetJob(jobID string) (map[string]string, error) {
	key := "job:" + jobID
	return s.rdb.HGetAll(ctx, key).Result()
}
