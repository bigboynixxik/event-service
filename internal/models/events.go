package models

import (
	"time"

	"github.com/google/uuid"
)

type EventStatus string

const (
	StatusDraft     EventStatus = "draft"
	StatusActive    EventStatus = "active"
	StatusCancelled EventStatus = "cancelled"
	StatusCompleted EventStatus = "completed"
)

type Events struct {
	ID              uuid.UUID   `json:"id""`
	CreatorID       uuid.UUID   `json:"creator_id"`
	IsPrivate       bool        `json:"is_private"`
	Title           string      `json:"title"`
	Description     *string     `json:"description,omitempty"`
	StartsAt        time.Time   `json:"starts_at"`
	DurationMinutes int         `json:"duration_minutes"`
	LocationName    *string     `json:"location_name,omitempty"`
	LocationCoords  *string     `json:"location_coords,omitempty"`
	MaxParticipants *int        `json:"max_participants,omitempty"`
	Status          EventStatus `json:"status"`
	EventCode       int         `json:"event_code"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type UpdateEventParams struct {
	Title           *string      `json:"title"`
	Description     *string      `json:"description"`
	IsPrivate       *bool        `json:"is_private"`
	StartsAt        *time.Time   `json:"starts_at"`
	DurationMinutes *int         `json:"duration_minutes"`
	LocationName    *string      `json:"location_name"`
	LocationCoords  *string      `json:"location_coords"`
	MaxParticipants *int         `json:"max_participants"`
	Status          *EventStatus `json:"status"`
}
