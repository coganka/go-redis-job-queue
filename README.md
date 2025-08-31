# Redis Job Queue (Go)

A minimal **distributed job queue** built in **Go** with **Redis**.
Supports **status tracking, retries with exponential backoff, scheduled jobs, and a dead-letter queue (DLQ)**.
Inspired by systems like Sidekiq / Celery, simplified for learning and portfolio.

---

## ✨ Features

* REST API to enqueue jobs (`queued` / `scheduled`)
* Worker pool (goroutines) for concurrent job processing
* Status tracking: `queued → processing → succeeded / retrying / failed`
* Automatic retries with exponential backoff
* Dead-letter queue (DLQ) for exhausted jobs
* Scheduled jobs (`scheduled_at`) via Redis Sorted Sets
* Docker + docker-compose support for easy setup

---

## 🏗 Architecture

```
API  --->  Redis Streams / ZSETs  --->  Worker(s)
           |        |                   |
           |        |---- RetryMgr <----|
           |---- Scheduler <------------|
           |---- DLQ ------------------>|
```

* **Streams**: main job pipeline
* **ZSET**: scheduled and retry queues
* **Scheduler**: moves due jobs into stream
* **Retry Manager**: re-enqueues failed jobs with delay
* **DLQ**: stores jobs that exhausted retries

---

## 🚀 Running with Docker

Make sure you have Docker + docker-compose installed.

```bash
git clone https://github.com/coganka/go-redis-job-queue.git
cd redis-job-queue
docker-compose up --build
```

Services:

* **API** → [http://localhost:8080](http://localhost:8080)
* **Worker** → runs in background
* **Redis** → port 6379

---

## 📡 API Usage

### Health Check

```bash
curl localhost:8080/healthz | jq
```

### Enqueue a Job

```bash
curl -X POST localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type":"echo.process","payload":{"msg":"hello"}}' | jq
```

### Enqueue a Scheduled Job (+15 seconds)

```bash
curl -X POST localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{"type":"echo.process","payload":{"msg":"delayed"},"scheduled_at":'$(date -v+15S +%s)'}' | jq
```

### Get Job Status

```bash
curl localhost:8080/jobs/<job-id> | jq
```

Example response:

```json
{
  "status": "succeeded",
  "created_at": "1735503200",
  "started_at": "1735503201",
  "finished_at": "1735503202",
  "attempts": "1",
  "updated_at": "1735503202"
}
```

### View Dead Letter Queue

```bash
curl localhost:8080/dlq | jq
```

---

## ⚙️ Project Structure

```
cmd/
 ├── api/       # API service (REST endpoints)
 └── worker/    # Worker service (job processor)

internal/
 ├── config/    # Env + configuration
 ├── queue/     # Redis queue, worker, retry manager, scheduler
 └── store/     # Job status persistence
```

---

## 📖 Tech Stack

* **Go 1.22**
* **Redis 7**
* **Docker + Compose**
* **Gin** (REST API)
* **go-redis** (Redis client)

---

## 📝 Notes

* This project is for **learning + portfolio**.
* Not production-ready (no auth, scaling, persistence tuning).
* Shows backend concepts: **concurrency, retries, scheduling, DLQ**.

---
