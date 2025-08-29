package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"redis-job-queue/internal/store"

	"github.com/redis/go-redis/v9"
)

func StartScheduler(rdb *redis.Client, stream string, s *store.Store) {
	go func() {
		ctx := context.Background()

		for {
			now := time.Now().Unix()
			jobs, err := rdb.ZRangeByScore(ctx, "jobs:scheduled", &redis.ZRangeBy{
				Min:   "-inf",
				Max:   fmt.Sprint(now),
				Count: 10,
			}).Result()

			if err != nil && err != redis.Nil {
				log.Printf("[scheduler] error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, raw := range jobs {
				var job JobEnvelope
				if err := json.Unmarshal([]byte(raw), &job); err != nil {
					log.Printf("[scheduler] bad job json: %v", err)
					_, _ = rdb.ZRem(ctx, "jobs:scheduled", raw).Result()
					continue
				}

				// push into main stream
				_, err := rdb.XAdd(ctx, &redis.XAddArgs{
					Stream: stream,
					Values: map[string]interface{}{"job": raw},
				}).Result()
				if err != nil {
					log.Printf("[scheduler] XADD failed: %v", err)
					continue
				}

				// remove from scheduled
				_, _ = rdb.ZRem(ctx, "jobs:scheduled", raw).Result()

				// mark status
				_ = s.SetStatus(job.ID, "queued", map[string]interface{}{
					"released_at": time.Now().Unix(),
				})

				log.Printf("[scheduler] released scheduled job %s", job.ID)
			}

			time.Sleep(1 * time.Second)
		}
	}()
}
