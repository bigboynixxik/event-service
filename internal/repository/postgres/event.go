package postgres

import (
	"context"
	"errors"
	"eventify-events/internal/models"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct {
	db      *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

var eventColumns = []string{"id", "creator_id", "is_private", "title", "description", "starts_at", "duration_minutes", "location_name", "location_coords", "max_participants", "status", "event_code"}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{
		db:      db,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *EventRepository) Create(ctx context.Context, e models.Events) error {
	sql, args, err := r.builder.Insert("events").
		Columns(eventColumns...).
		Values(e.ID,
			e.CreatorID,
			e.IsPrivate,
			e.Title,
			e.Description,
			e.StartsAt,
			e.DurationMinutes,
			e.LocationName,
			e.LocationCoords,
			e.MaxParticipants,
			e.Status,
			e.EventCode).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query %w", err)
	}
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("failed to intsert: %w", err)
	}
	return nil
}

func (r *EventRepository) GetEvent(ctx context.Context, id uuid.UUID) (models.Events, error) {
	sql, args, err := r.builder.Select(eventColumns...).
		From("events").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return models.Events{}, fmt.Errorf("failde to build query: %w", err)
	}

	var e models.Events

	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&e.ID,
		&e.CreatorID,
		&e.IsPrivate,
		&e.Title,
		&e.Description,
		&e.StartsAt,
		&e.DurationMinutes,
		&e.LocationName,
		&e.LocationCoords,
		&e.MaxParticipants,
		&e.Status,
		&e.EventCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Events{}, fmt.Errorf("failed to get event: %w", err)
		}
	}
	return e, nil
}
