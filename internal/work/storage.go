package work

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	FetchTemplates(ctx context.Context, taskID uuid.UUID, sectionID uuid.UUID) (templates map[uuid.UUID]*Template, err error)
	Stream(ctx context.Context, taskID uuid.UUID, sectionID uuid.UUID) (stream <-chan *Work, err error)
}
