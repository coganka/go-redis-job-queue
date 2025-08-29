package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"redis-job-queue/internal/config"
	"redis-job-queue/internal/queue"
	"redis-job-queue/internal/store"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	q := queue.NewRedisClient(cfg)
	s := store.New(q.Client())

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// enqueue
	r.POST("/jobs", func(c *gin.Context) {
		var req struct {
			Type        string          `json:"type" binding:"required"`
			Payload     json.RawMessage `json:"payload" binding:"required"`
			ScheduledAt int64           `json:"scheduled_at"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := q.Enqueue(req.Type, req.Payload, req.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if req.ScheduledAt > time.Now().Unix() {
			_ = s.SetStatus(id, "scheduled", map[string]interface{}{
				"created_at":   time.Now().Unix(),
				"scheduled_at": req.ScheduledAt,
			})

		} else {
			_ = s.SetStatus(id, "queued", map[string]interface{}{
				"created_at": time.Now().Unix(),
			})

		}

		c.JSON(http.StatusAccepted, gin.H{"id": id, "status": "accepted"})
	})

	// get status
	r.GET("/jobs/:id", func(c *gin.Context) {
		jobID := c.Param("id")
		data, err := s.GetJob(jobID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if len(data) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusOK, data)
	})

	r.GET("/dlq", func(c *gin.Context) {
		ctx := context.Background()

		// read from DLQ stream
		entries, err := q.Client().XRange(ctx, "jobs:dlq", "-", "+").Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		jobs := []map[string]string{}

		for _, e := range entries {
			result := make(map[string]string)
			for k, v := range e.Values {
				result[k] = fmt.Sprint(v) // convert any type to string
			}
			jobs = append(jobs, result)
		}

		c.JSON(http.StatusOK, jobs)
	})

	r.Run(":" + cfg.Port)
}
