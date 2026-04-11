package postgres_test

import (
	"context"
	"testing"
	"time"

	"eventify-events/internal/models"
	"eventify-events/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Ptr[T any](v T) *T { return &v }

func TestEventRepository_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	dbURL := "postgres://user:password@localhost:5432/postgres?sslmode=disable"

	pool, err := postgres.NewPool(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.NewEventRepository(pool)

	_, err = pool.Exec(ctx, "DELETE FROM events")
	require.NoError(t, err)

	eventID := uuid.New()
	testEvent := models.Events{
		ID:        eventID,
		CreatorID: uuid.New(),
		Title:     "Test Event",
		EventCode: uuid.New().String()[:8], // Уникальный код
		StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
		Status:    models.StatusDraft,
	}

	t.Run("Create event", func(t *testing.T) {
		err := repo.CreateEvent(ctx, testEvent)
		assert.NoError(t, err)
	})

	t.Run("Get existing event", func(t *testing.T) {
		found, err := repo.GetEvent(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, testEvent.ID, found.ID)
		assert.Equal(t, testEvent.Title, found.Title)
		assert.Equal(t, testEvent.Status, found.Status)
	})

	t.Run("Get non-existent event", func(t *testing.T) {
		_, err := repo.GetEvent(ctx, uuid.New())
		assert.Error(t, err)
	})

	t.Run("Get all events", func(t *testing.T) {
		_, err := repo.ListEvents(ctx)
		assert.NoError(t, err)
	})

	t.Run("Update event - success", func(t *testing.T) {
		id := uuid.New()
		initialEvent := models.Events{
			ID:          id,
			CreatorID:   uuid.New(),
			Title:       "Old Title",
			EventCode:   uuid.New().String()[:8],
			Description: Ptr("Old Description"),
			StartsAt:    time.Now().Add(time.Hour).Truncate(time.Second),
			Status:      models.StatusDraft,
		}
		require.NoError(t, repo.CreateEvent(ctx, initialEvent))

		newTitle := "Updated Title"
		params := models.UpdateEventParams{
			Title: &newTitle,
		}

		updated, err := repo.UpdateEvent(ctx, params, id)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, updated.Title)
		assert.Equal(t, initialEvent.Status, updated.Status)
		assert.Equal(t, *initialEvent.Description, *updated.Description)
		assert.True(t, updated.UpdatedAt.After(initialEvent.UpdatedAt))
	})

	t.Run("Update event - non-existent ID", func(t *testing.T) {
		newTitle := "Nobody Home"
		params := models.UpdateEventParams{Title: &newTitle}

		_, err := repo.UpdateEvent(ctx, params, uuid.New())
		assert.Error(t, err)
	})

	t.Run("Update event - empty params", func(t *testing.T) {
		id := uuid.New()
		event := models.Events{
			ID:        id,
			CreatorID: uuid.New(),
			Title:     "Constant Event",
			EventCode: uuid.New().String()[:8],
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		}
		require.NoError(t, repo.CreateEvent(ctx, event))

		params := models.UpdateEventParams{}
		updated, err := repo.UpdateEvent(ctx, params, id)

		assert.NoError(t, err)
		assert.Equal(t, event.Title, updated.Title)
		assert.Equal(t, event.Status, updated.Status)
	})
	t.Run("Join event - success", func(t *testing.T) {
		eID := uuid.New()
		event := models.Events{
			ID:        eID,
			CreatorID: uuid.New(),
			Title:     "Event for Joining",
			EventCode: uuid.New().String()[:8],
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		}
		require.NoError(t, repo.CreateEvent(ctx, event))

		userID := uuid.New()
		returnedID, ok, err := repo.JoinEvent(ctx, userID, eID)

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, eID, returnedID)
	})

	t.Run("Join event - duplicate join", func(t *testing.T) {

		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Unique Event", EventCode: uuid.New().String()[:8],
			StartsAt: time.Now().Add(time.Hour).Truncate(time.Second), Status: models.StatusDraft,
		}))

		userID := uuid.New()

		_, ok1, err1 := repo.JoinEvent(ctx, userID, eID)
		assert.NoError(t, err1)
		assert.True(t, ok1)

		_, ok2, err2 := repo.JoinEvent(ctx, userID, eID)
		assert.Error(t, err2) // База должна выкинуть Unique Violation
		assert.False(t, ok2)
	})

	t.Run("Join event - non-existent event", func(t *testing.T) {
		userID := uuid.New()
		fakeEventID := uuid.New()

		_, ok, err := repo.JoinEvent(ctx, userID, fakeEventID)

		assert.Error(t, err)
		assert.False(t, ok)
	})
}
