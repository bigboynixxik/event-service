package repository

import (
	"context"
	"eventify-events/internal/models"

	"github.com/google/uuid"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, event models.Events) error
	GetEvent(ctx context.Context, id uuid.UUID) (models.Events, error)
	ListUserEvents(ctx context.Context, userId uuid.UUID) ([]models.Events, error)
	ListEvents(ctx context.Context) ([]models.Events, error)
	UpdateEvent(ctx context.Context, params models.UpdateEventParams, id uuid.UUID) (models.Events, error)
	JoinEvent(ctx context.Context, userId uuid.UUID, eventId uuid.UUID) (uuid.UUID, bool, error)
}
