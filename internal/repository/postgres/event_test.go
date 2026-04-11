package postgres_test

import (
	"context"
	postgres2 "eventify-events/pkg/postgres"
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

	pool, err := postgres2.NewPool(ctx, dbURL)
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
		returnedID, ok, err := repo.AddParticipant(ctx, userID, eID)

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

		_, ok1, err1 := repo.AddParticipant(ctx, userID, eID)
		assert.NoError(t, err1)
		assert.True(t, ok1)

		_, ok2, err2 := repo.AddParticipant(ctx, userID, eID)
		assert.NoError(t, err2)
		assert.False(t, ok2)
	})

	t.Run("Join event - non-existent event", func(t *testing.T) {
		userID := uuid.New()
		fakeEventID := uuid.New()

		_, ok, err := repo.AddParticipant(ctx, userID, fakeEventID)

		assert.Error(t, err)
		assert.False(t, ok)
	})
	t.Run("Leave event - success", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Exit Event", EventCode: uuid.New().String()[:8],
			StartsAt: time.Now().Add(time.Hour).Truncate(time.Second), Status: models.StatusDraft,
		}))

		uID := uuid.New()
		_, joined, err := repo.AddParticipant(ctx, uID, eID)
		require.NoError(t, err)
		require.True(t, joined)

		left, err := repo.RemoveParticipant(ctx, uID, eID)

		assert.NoError(t, err)
		assert.True(t, left)
	})

	t.Run("Leave event - not a participant", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "No Member Event", EventCode: uuid.New().String()[:8],
			StartsAt: time.Now().Add(time.Hour).Truncate(time.Second), Status: models.StatusDraft,
		}))

		uID := uuid.New()

		left, err := repo.RemoveParticipant(ctx, uID, eID)

		assert.NoError(t, err)
		assert.False(t, left)
	})

	t.Run("Leave event - twice", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Double Exit", EventCode: uuid.New().String()[:8],
			StartsAt: time.Now().Add(time.Hour).Truncate(time.Second), Status: models.StatusDraft,
		}))

		uID := uuid.New()
		_, _, _ = repo.AddParticipant(ctx, uID, eID)

		left1, _ := repo.RemoveParticipant(ctx, uID, eID)
		assert.True(t, left1)

		left2, err := repo.RemoveParticipant(ctx, uID, eID)
		assert.NoError(t, err)
		assert.False(t, left2)
	})
	t.Run("Get participants - success", func(t *testing.T) {
		eID := uuid.New()
		event := models.Events{
			ID:        eID,
			CreatorID: uuid.New(),
			Title:     "Participants Test",
			EventCode: uuid.New().String()[:8],
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		}
		err := repo.CreateEvent(ctx, event)
		require.NoError(t, err)

		uID1 := uuid.New()
		uID2 := uuid.New()

		_, ok1, err1 := repo.AddParticipant(ctx, uID1, eID)
		require.NoError(t, err1)
		require.True(t, ok1)

		_, ok2, err2 := repo.AddParticipant(ctx, uID2, eID)
		require.NoError(t, err2)
		require.True(t, ok2)

		list, err := repo.GetEventParticipants(ctx, eID)
		assert.NoError(t, err)
		assert.Len(t, list, 2)
	})
	t.Run("Join event by code - success", func(t *testing.T) {
		eID := uuid.New()
		myCode := "TOP-SECRET"
		event := models.Events{
			ID:        eID,
			CreatorID: uuid.New(),
			Title:     "Code Party",
			EventCode: myCode,
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		}
		require.NoError(t, repo.CreateEvent(ctx, event))

		uID := uuid.New()
		joined, err := repo.JoinEvent(ctx, uID, myCode)

		assert.NoError(t, err)
		assert.True(t, joined)

		participants, err := repo.GetEventParticipants(ctx, eID)
		assert.NoError(t, err)
		assert.Len(t, participants, 1)
		assert.Equal(t, uID, participants[0].UserID)
	})

	t.Run("Join event by code - wrong code", func(t *testing.T) {
		uID := uuid.New()

		joined, err := repo.JoinEvent(ctx, uID, "WRONG-CODE-123")

		assert.NoError(t, err)
		assert.False(t, joined)
	})

	t.Run("Join event by code - already joined", func(t *testing.T) {
		eID := uuid.New()
		myCode := "DOUBLE-JOIN"

		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Busy Event", EventCode: myCode,
			StartsAt: time.Now().Add(time.Hour).Truncate(time.Second), Status: models.StatusDraft,
		}))

		uID := uuid.New()

		ok1, err1 := repo.JoinEvent(ctx, uID, myCode)
		require.NoError(t, err1)
		assert.True(t, ok1)

		ok2, err2 := repo.JoinEvent(ctx, uID, myCode)

		assert.NoError(t, err2)
		assert.False(t, ok2)

		participants, _ := repo.GetEventParticipants(ctx, eID)
		assert.Len(t, participants, 1)
	})
}
