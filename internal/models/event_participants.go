package models

import (
	"time"

	"github.com/google/uuid"
)

type EventParticipantsStatus string

const (
	StatusInvited   EventParticipantsStatus = "invited"
	StatusConfirmed EventParticipantsStatus = "confirmed"
	StatusDeclined  EventParticipantsStatus = "declined"
	StatusMaybe     EventParticipantsStatus = "maybe"
)

type EventParticipants struct {
	ID                    uuid.UUID               `json:"id" db:"id"`
	UserID                uuid.UUID               `json:"user_id" db:"user_id"`
	EventID               uuid.UUID               `json:"event_id" db:"event_id"`
	IsOwner               bool                    `json:"is_owner" db:"is_owner"`
	CanEditEvent          bool                    `json:"can_edit_event" db:"can_edit_event"`
	CanManageParticipants bool                    `json:"can_manage_participants" db:"can_manage_participants"`
	CanManageChecklist    bool                    `json:"can_manage_checklist" db:"can_manage_checklist"`
	Role                  *string                 `json:"role,omitempty" db:"role"`
	Status                EventParticipantsStatus `json:"status" db:"status"`
	JoinedAt              time.Time               `json:"joined_at" db:"joined_at"`
	LeftAt                *time.Time              `json:"left_at,omitempty" db:"left_at"`
}

func (e *EventParticipants) Values() []any {
	return []any{
		e.ID, e.UserID, e.EventID, e.IsOwner, e.CanEditEvent,
		e.CanManageParticipants, e.CanManageChecklist, e.Role, e.Status,
		e.JoinedAt, e.LeftAt,
	}
}
