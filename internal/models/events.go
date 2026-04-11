package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type EventStatus string

const (
	StatusDraft     EventStatus = "draft"
	StatusActive    EventStatus = "active"
	StatusCancelled EventStatus = "cancelled"
	StatusCompleted EventStatus = "completed"
)

type Events struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	CreatorID       uuid.UUID       `json:"creator_id" db:"creator_id"`
	IsPrivate       bool            `json:"is_private" db:"is_private"`
	Title           string          `json:"title" db:"title"`
	Description     *string         `json:"description,omitempty" db:"description"`
	StartsAt        time.Time       `json:"starts_at" db:"starts_at"`
	Duration        pgtype.Interval `json:"duration" db:"duration"`
	LocationName    *string         `json:"location_name,omitempty" db:"location_name"`
	LocationCoords  *pgtype.Point   `json:"location_coords,omitempty" db:"location_coords"`
	MaxParticipants *int            `json:"max_participants,omitempty" db:"max_participants"`
	Status          EventStatus     `json:"status" db:"status"`
	EventCode       string          `json:"event_code" db:"event_code"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

type UpdateEventParams struct {
	Title          *string       `json:"title" db:"title"`
	Description    *string       `json:"description" db:"description"`
	StartsAt       *time.Time    `json:"starts_at" db:"starts_at"`
	LocationName   *string       `json:"location_name" db:"location_name"`
	LocationCoords *pgtype.Point `json:"location_coords" db:"location_coords"`
}

func (e *Events) Values() []any {
	return []any{
		e.ID, e.CreatorID, e.IsPrivate, e.Title, e.Description,
		e.StartsAt, e.Duration, e.LocationName, e.LocationCoords,
		e.MaxParticipants, e.Status, e.EventCode, e.CreatedAt, e.UpdatedAt,
	}
}
