# ---- Build stage ----
FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# build api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api
# build worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker

# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /bin/api /bin/api
COPY --from=builder /bin/worker /bin/worker

# default port for API
EXPOSE 8080

ENTRYPOINT ["/bin/api"]
    