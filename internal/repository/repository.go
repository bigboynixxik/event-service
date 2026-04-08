package repository

import (
	"context"
	"eventify-events/internal/models"

	"github.com/google/uuid"
)

type EventRepository interface {
	Create(ctx context.Context, event models.Events) error
	GetEvent(ctx context.Context, id uuid.UUID) (models.Events, error)
}
