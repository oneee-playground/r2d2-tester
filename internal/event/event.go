package event

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const Topic = "test"

type TestEvent struct {
	ID      uuid.UUID     `json:"id"`
	Success bool          `json:"success"`
	Took    time.Duration `json:"took"`
	Extra   string        `json:"extra"`
}

type Publisher interface {
	Publish(ctx context.Context, e TestEvent) error
}
