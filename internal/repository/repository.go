package repository

import (
	"context"
	"eventify-events/internal/models"
	"time"

	"github.com/google/uuid"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, e models.Events) error
	GetEvent(ctx context.Context, id uuid.UUID) (models.Events, error)
	ListUserEvents(ctx context.Context, userId uuid.UUID) ([]models.Events, error)
	ListEvents(ctx context.Context) ([]models.Events, error)
	UpdateEvent(ctx context.Context, params models.UpdateEventParams, id uuid.UUID) (models.Events, error)
	GetEventByCode(ctx context.Context, code string) (models.Events, error)
	JoinEvent(ctx context.Context, userId uuid.UUID, eventId uuid.UUID, isOwner bool) (uuid.UUID, bool, error)
	RemoveParticipant(ctx context.Context, participantId uuid.UUID, eventId uuid.UUID) (bool, error)
	GetEventParticipants(ctx context.Context, eventId uuid.UUID) ([]models.EventParticipants, error)
	CancelEvent(ctx context.Context, eventId uuid.UUID) (bool, error)
	CreateInviteLink(ctx context.Context, eventId uuid.UUID, inviteType string, expiresAt *time.Time) (string, error)
	AddChecklistItem(ctx context.Context, e models.ChecklistItems) (uuid.UUID, error)
	GetEventChecklist(ctx context.Context, eventId uuid.UUID) ([]models.ChecklistItems, error)
	RemoveChecklistItem(ctx context.Context, itemId uuid.UUID, eventId uuid.UUID) (bool, error)
	MarkItemPurchased(ctx context.Context, eventId uuid.UUID, itemId uuid.UUID, buyerId *uuid.UUID, isPurchased *bool) (bool, error)
	GetInviteByToken(ctx context.Context, token string) (models.EventInvites, error)
	UseInvite(ctx context.Context, inviteId uuid.UUID) (bool, error)
	GetParticipant(ctx context.Context, userId uuid.UUID, eventId uuid.UUID) (models.EventParticipants, error)
}
