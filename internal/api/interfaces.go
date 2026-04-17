package api

import (
	"context"
	"eventify-events/internal/models"
	"time"
	"github.com/google/uuid"
	"eventify-events/internal/services"
)
type EventServiceInterface interface {
    GetEvent(ctx context.Context, id uuid.UUID) (*models.Events, error)
    ListEvents(ctx context.Context, filter services.ListEventsFilter) ([]models.Events, error)
    ListUserEvents(ctx context.Context, userID uuid.UUID) ([]models.Events, error)
    CreateEvent(ctx context.Context, callerID uuid.UUID, eventParams services.EventInputParams) (models.Events, error)
    UpdateEvent(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, params models.UpdateEventParams) (models.Events, error)
    JoinEvent(ctx context.Context, userID uuid.UUID, code string) (bool, error)
    LeaveEvent(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID) (bool, error)
    RemoveParticipant(ctx context.Context, callerID uuid.UUID, participantID uuid.UUID, eventID uuid.UUID) (bool, error)
    GetEventParticipants(ctx context.Context, eventID uuid.UUID) ([]models.EventParticipants, error)
    CancelEvent(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID) (bool, error)
    CreateInviteLink(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, inviteType string, expiresAt *time.Time) (string, error)
}

type ChecklistServiceInterface interface {
    AddChecklistItem(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, title string, quantity int, unit string) (uuid.UUID, error)
    RemoveChecklistItem(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, itemID uuid.UUID) (bool, error)
    MarkItemPurchased(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, itemID uuid.UUID, buyerID *uuid.UUID, isPurchased *bool) (bool, error)
    GetEventChecklist(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID) ([]models.ChecklistItems, error)
}