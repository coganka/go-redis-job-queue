# Redis Job Queue

An asynchronous job queue in Go with concurrent workers, delayed execution, retries, and a dead-letter queue. A small HTTP API accepts work and exposes its state; a separate worker process handles execution.

**Go · Redis Streams · Redis Sorted Sets · Gin**

## How a job moves through the system

```text
POST /jobs ── immediate ────────────→ Redis Stream → Worker pool
      │                                   ↑              │
      └── scheduled → Sorted Set → Scheduler             ├── success → status
                                          ↑              ├── retry → Sorted Set
                                    Retry manager ───────┘
                                                         └── exhausted → DLQ
```

Streams provide consumer-group reads and acknowledgements. Sorted sets hold scheduled jobs and retries, using execution time as the score. Job metadata is stored separately in Redis hashes so clients can query progress without reading the queue.

## What is implemented

- Three worker goroutines in the default worker process.
- Immediate and scheduled jobs through `POST /jobs`.
- Status transitions including `queued`, `scheduled`, `processing`, `retrying`, `succeeded`, and `failed`.
- Exponential-backoff retries with jitter and a default maximum of five attempts.
- A dead-letter stream for jobs that exhaust their retries.
- Separate API and worker entry points.

The included `echo.process` handler simulates a one-second task. Unknown job types exercise the retry and failure path.

## Run locally

Use **Go 1.23.2 or newer**, Redis 7+, and two terminals. Start Redis locally before launching either process.

```bash
git clone https://github.com/coganka/go-redis-job-queue.git
cd go-redis-job-queue
go mod download
```

Terminal 1:

```bash
REDIS_ADDR=localhost:6379 PORT=8080 go run ./cmd/api
```

Terminal 2:

```bash
REDIS_ADDR=localhost:6379 go run ./cmd/worker
```

Configuration is read from process environment variables; `.env` is not loaded by the Go application itself.

| Variable | Default |
|---|---|
| `REDIS_ADDR` | `localhost:6379` |
| `REDIS_DB` | `0` |
| `STREAM` | `jobs:stream` |
| `CONSUMER_GROUP` | `jobs:cg` |
| `PORT` | `8080` |

`API_KEY` exists in configuration but is not enforced by the HTTP handlers.

## Try the job lifecycle

```bash
curl -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"echo.process","payload":{"message":"hello"}}'
```

The API returns `202` with a job ID. Use that ID to retrieve its status:

```bash
curl http://localhost:8080/jobs/YOUR_JOB_ID
```

To schedule a job, include `scheduled_at` as a future Unix timestamp in seconds. To see retries and the dead-letter queue, submit an unsupported job type:

```bash
curl -X POST http://localhost:8080/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"demo.unsupported","payload":{}}'

curl http://localhost:8080/dlq
```

Wait for retries to finish before checking the DLQ. The default backoff takes roughly half a minute, plus scheduler timing and processing overhead.

## Read the implementation

Start with [worker.go](internal/queue/worker.go) for execution, retries, and acknowledgements. [scheduler.go](internal/queue/scheduler.go) and [retry.go](internal/queue/retry.go) move due jobs into the stream; [store.go](internal/store/store.go) persists status.

## Current boundaries

The native commands above are the supported path documented here. The checked-in Docker files need alignment: the builder uses Go 1.22, the Compose file lacks a Redis service and a modern `services` wrapper, and the worker needs an entrypoint override rather than a command argument to the API entrypoint.

The queue demonstrates execution and retry mechanics, but does not yet guarantee recovery after worker crashes. Pending-message reclamation, idempotent handlers, and atomic queue transitions are follow-up work. In particular, the worker can acknowledge a message after a failed retry/DLQ write, and multiple scheduler instances can release the same job. The timeout field is metadata only; execution deadlines are not enforced.
