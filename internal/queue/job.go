package queue

import "encoding/json"

type JobEnvelope struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"max_attempts"`
	TimeoutMS   int             `json:"timeout_ms"`
	CreatedAt   int64           `json:"created_at"`
	ScheduledAt int64           `json:"scheduled_at,omitempty"`
}
