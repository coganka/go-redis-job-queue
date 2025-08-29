package main

import (
	"log"

	"redis-job-queue/internal/config"
	"redis-job-queue/internal/queue"
	"redis-job-queue/internal/store"
)

func main() {
	cfg := config.Load()
	q := queue.NewRedisClient(cfg)
	s := store.New(q.Client())

	worker := queue.NewWorker(q, "worker-1", 3, s)
	worker.Start()

	// start retry manager
	queue.StartRetryManager(q.Client(), cfg.Stream, s)
	queue.StartScheduler(q.Client(), cfg.Stream, s)

	log.Println("Worker running. Press Ctrl+C to exit.")
	select {}
}
