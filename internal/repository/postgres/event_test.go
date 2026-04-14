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
	testCode := uuid.New().String()[:8]
	testEvent := models.Events{
		ID:        eventID,
		CreatorID: uuid.New(),
		Title:     "Test Event",
		EventCode: testCode,
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
		params := models.UpdateEventParams{Title: &newTitle}
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
		returnedID, ok, err := repo.JoinEvent(ctx, userID, eID, false)

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
		_, ok1, err1 := repo.JoinEvent(ctx, userID, eID, false)
		assert.NoError(t, err1)
		assert.True(t, ok1)

		_, ok2, err2 := repo.JoinEvent(ctx, userID, eID, false)
		assert.NoError(t, err2)
		assert.False(t, ok2)
	})

	t.Run("Join event - non-existent event", func(t *testing.T) {
		userID := uuid.New()
		fakeEventID := uuid.New()
		_, ok, err := repo.JoinEvent(ctx, userID, fakeEventID, false)
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
		_, joined, err := repo.JoinEvent(ctx, uID, eID, false)
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
		_, _, _ = repo.JoinEvent(ctx, uID, eID, false)

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

		_, ok1, err1 := repo.JoinEvent(ctx, uID1, eID, false)
		require.NoError(t, err1)
		require.True(t, ok1)

		_, ok2, err2 := repo.JoinEvent(ctx, uID2, eID, false)
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
		found, err := repo.GetEventByCode(ctx, myCode)
		assert.NoError(t, err)

		_, joined, err := repo.JoinEvent(ctx, uID, found.ID, false)
		assert.NoError(t, err)
		assert.True(t, joined)

		participants, err := repo.GetEventParticipants(ctx, eID)
		assert.NoError(t, err)
		assert.Len(t, participants, 1)
		assert.Equal(t, uID, participants[0].UserID)
	})

	t.Run("Join event by code - wrong code", func(t *testing.T) {
		_, err := repo.GetEventByCode(ctx, "WRONG-CODE-123")
		assert.Error(t, err)
	})

	t.Run("Join event by code - already joined", func(t *testing.T) {
		eID := uuid.New()
		myCode := "DOUBLE-JOIN"
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Busy Event", EventCode: myCode,
			StartsAt: time.Now().Add(time.Hour).Truncate(time.Second), Status: models.StatusDraft,
		}))

		uID := uuid.New()
		found, _ := repo.GetEventByCode(ctx, myCode)

		_, ok1, err1 := repo.JoinEvent(ctx, uID, found.ID, false)
		assert.NoError(t, err1)
		assert.True(t, ok1)

		_, ok2, err2 := repo.JoinEvent(ctx, uID, found.ID, false)
		assert.NoError(t, err2)
		assert.False(t, ok2)

		participants, _ := repo.GetEventParticipants(ctx, eID)
		assert.Len(t, participants, 1)
	})

	t.Run("Cancel event - success", func(t *testing.T) {
		id := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID:        id,
			CreatorID: uuid.New(),
			Title:     "To Be Cancelled",
			EventCode: "CANCEL-ME",
			Status:    models.StatusDraft,
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		ok, err := repo.CancelEvent(ctx, id)
		assert.NoError(t, err)
		assert.True(t, ok)

		updated, err := repo.GetEvent(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, models.StatusCancelled, updated.Status)
	})

	t.Run("Create invite link - success", func(t *testing.T) {
		eID := uuid.New()
		expectedCode := "SECRET-LINK-123"
		err := repo.CreateEvent(ctx, models.Events{
			ID:        eID,
			CreatorID: uuid.New(),
			Title:     "Invite Only Event",
			EventCode: expectedCode,
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		})
		require.NoError(t, err)

		inviteType := "single"
		expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Second)
		returnedCode, err := repo.CreateInviteLink(ctx, eID, inviteType, &expiresAt)

		assert.NoError(t, err)
		assert.Equal(t, expectedCode, returnedCode)
	})

	t.Run("Create invite link - event not found", func(t *testing.T) {
		fakeID := uuid.New()
		code, err := repo.CreateInviteLink(ctx, fakeID, "unlimited", nil)
		assert.Error(t, err)
		assert.Empty(t, code)
	})

	t.Run("Check multi invite in DB", func(t *testing.T) {
		eID := uuid.New()
		err := repo.CreateEvent(ctx, models.Events{
			ID:        eID,
			CreatorID: uuid.New(),
			Title:     "Multi Invite Test",
			EventCode: "MULTI-123",
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		})
		require.NoError(t, err)

		_, err = repo.CreateInviteLink(ctx, eID, "multi", nil)
		assert.NoError(t, err)

		var inviteType string
		query := "SELECT invite_type FROM event_invites WHERE event_id = $1"
		err = pool.QueryRow(ctx, query, eID).Scan(&inviteType)
		assert.NoError(t, err)
		assert.Equal(t, "multi", inviteType)
	})

	t.Run("Add and get multiple checklist items", func(t *testing.T) {
		eID := uuid.New()
		err := repo.CreateEvent(ctx, models.Events{
			ID:        eID,
			CreatorID: uuid.New(),
			Title:     "Party List",
			EventCode: "LIST-1",
			StartsAt:  time.Now().Add(time.Hour).Truncate(time.Second),
			Status:    models.StatusDraft,
		})
		require.NoError(t, err)

		item1 := models.ChecklistItems{ID: uuid.New(), EventID: eID, Title: "Water"}
		_, err = repo.AddChecklistItem(ctx, item1)
		require.NoError(t, err)

		item2 := models.ChecklistItems{ID: uuid.New(), EventID: eID, Title: "Meat"}
		_, err = repo.AddChecklistItem(ctx, item2)
		require.NoError(t, err)

		list, err := repo.GetEventChecklist(ctx, eID)
		assert.NoError(t, err)
		assert.Len(t, list, 2)
	})

	t.Run("Get checklist - empty list", func(t *testing.T) {
		list, err := repo.GetEventChecklist(ctx, uuid.New())
		assert.NoError(t, err)
		assert.Len(t, list, 0)
	})

	t.Run("Remove checklist item - success", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Delete Test", Status: models.StatusDraft,
			EventCode: "DEL-1", StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		itemID, err := repo.AddChecklistItem(ctx, models.ChecklistItems{ID: uuid.New(), EventID: eID, Title: "Item"})
		require.NoError(t, err)

		removed, err := repo.RemoveChecklistItem(ctx, itemID, eID)
		assert.NoError(t, err)
		assert.True(t, removed)
	})

	t.Run("Mark item purchased - full update", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Purchase Test", Status: models.StatusDraft,
			EventCode: "PUR-1", StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		itemID, _ := repo.AddChecklistItem(ctx, models.ChecklistItems{ID: uuid.New(), EventID: eID, Title: "Milk"})
		pID := uuid.New()
		_, _, err := repo.JoinEvent(ctx, pID, eID, false)
		require.NoError(t, err)

		ok, err := repo.MarkItemPurchased(ctx, eID, itemID, &pID, Ptr(true))
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Mark item purchased - only status", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Stat Test", Status: models.StatusDraft,
			EventCode: "STAT-1", StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))
		itemID, _ := repo.AddChecklistItem(ctx, models.ChecklistItems{ID: uuid.New(), EventID: eID, Title: "Bread"})

		ok, err := repo.MarkItemPurchased(ctx, eID, itemID, nil, Ptr(true))
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Mark item purchased - only buyer", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Buyer Test", Status: models.StatusDraft,
			EventCode: "BUY-1", StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))
		itemID, _ := repo.AddChecklistItem(ctx, models.ChecklistItems{ID: uuid.New(), EventID: eID, Title: "Eggs"})
		pID := uuid.New()
		_, _, _ = repo.JoinEvent(ctx, pID, eID, false)

		ok, err := repo.MarkItemPurchased(ctx, eID, itemID, &pID, nil)
		assert.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("Leave event - owner cannot leave", func(t *testing.T) {
		eID := uuid.New()
		uID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uID, Title: "Owner Stay", EventCode: "STAY1",
			Status: models.StatusDraft, StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		_, _, _ = repo.JoinEvent(ctx, uID, eID, true)

		left, err := repo.RemoveParticipant(ctx, uID, eID)

		assert.NoError(t, err)
		assert.False(t, left)
	})
	t.Run("Invite: Get by token - success", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Invite Party", EventCode: "INV-123",
			Status: models.StatusDraft, StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		token := "TOP-SECRET-TOKEN"
		_, err := pool.Exec(ctx,
			"INSERT INTO event_invites (event_id, token, invite_type, max_uses, used_count) VALUES ($1, $2, $3, $4, $5)",
			eID, token, "single", 1, 0)
		require.NoError(t, err)

		found, err := repo.GetInviteByToken(ctx, token)
		assert.NoError(t, err)
		assert.Equal(t, eID, found.EventID)
		assert.Equal(t, token, found.Token)
	})

	t.Run("Invite: Get by token - not found", func(t *testing.T) {
		_, err := repo.GetInviteByToken(ctx, "GHOST-TOKEN")
		assert.Error(t, err)
	})
	t.Run("Invite: UseInvite - successful increment", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Increment Test", EventCode: "INC-1",
			Status: models.StatusDraft, StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		token := "MULTI-USE-TOKEN"
		var inviteID uuid.UUID
		err := pool.QueryRow(ctx,
			"INSERT INTO event_invites (event_id, token, invite_type, max_uses, used_count) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			eID, token, "multi", 10, 0).Scan(&inviteID)
		require.NoError(t, err)

		ok, err := repo.UseInvite(ctx, inviteID)
		assert.NoError(t, err)
		assert.True(t, ok)

		var count int
		err = pool.QueryRow(ctx, "SELECT used_count FROM event_invites WHERE id = $1", inviteID).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Invite: UseInvite - fail when limit reached", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Limit Test", EventCode: "LIM-1",
			Status: models.StatusDraft, StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		token := "SINGLE-USE-ONLY"
		var inviteID uuid.UUID
		err := pool.QueryRow(ctx,
			"INSERT INTO event_invites (event_id, token, invite_type, max_uses, used_count) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			eID, token, "single", 1, 1).Scan(&inviteID)
		require.NoError(t, err)

		ok, err := repo.UseInvite(ctx, inviteID)

		assert.NoError(t, err)
		assert.False(t, ok, "Should not be able to use invite when limit is reached")
	})

	t.Run("Invite: UseInvite - unlimited works", func(t *testing.T) {
		eID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eID, CreatorID: uuid.New(), Title: "Unlimited Test", EventCode: "UNL-1",
			Status: models.StatusDraft, StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		token := "FOREVER-TOKEN"
		var inviteID uuid.UUID

		query := "INSERT INTO event_invites (event_id, token, invite_type, max_uses, used_count) VALUES ($1, $2, $3, $4::int, $5) RETURNING id"

		err := pool.QueryRow(ctx, query, eID, token, "unlimited", nil, 0).Scan(&inviteID)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			ok, err := repo.UseInvite(ctx, inviteID)
			assert.NoError(t, err)
			assert.True(t, ok)
		}

		var count int
		pool.QueryRow(ctx, "SELECT used_count FROM event_invites WHERE id = $1", inviteID).Scan(&count)
		assert.Equal(t, 3, count)
	})
	t.Run("ListUserEvents - success combined", func(t *testing.T) {
		userID := uuid.New()
		otherID := uuid.New()

		desc1 := "User is the owner of this event"
		desc2 := "User is just a guest here"
		desc3 := "User has nothing to do with this"

		event1ID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID:          event1ID,
			CreatorID:   userID,
			Title:       "I am Creator",
			Description: &desc1,
			EventCode:   "OWNER123",
			StartsAt:    time.Now().Add(time.Hour).Truncate(time.Second),
			Status:      models.StatusDraft,
		}))

		event2ID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID:          event2ID,
			CreatorID:   otherID,
			Title:       "I am Participant",
			Description: &desc2,
			EventCode:   "GUEST456",
			StartsAt:    time.Now().Add(2 * time.Hour).Truncate(time.Second),
			Status:      models.StatusDraft,
		}))

		_, joined, err := repo.JoinEvent(ctx, userID, event2ID, false)
		require.NoError(t, err)
		require.True(t, joined)

		event3ID := uuid.New()
		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID:          event3ID,
			CreatorID:   otherID,
			Title:       "Stranger Event",
			Description: &desc3,
			EventCode:   "OTHER789",
			StartsAt:    time.Now().Add(3 * time.Hour).Truncate(time.Second),
			Status:      models.StatusDraft,
		}))

		events, err := repo.ListUserEvents(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, events, 2, "Должно вернуться ровно 2 ивента (где создатель и где участник)")

		var foundIDs []uuid.UUID
		for _, e := range events {
			foundIDs = append(foundIDs, e.ID)
		}

		assert.Contains(t, foundIDs, event1ID, "Список должен содержать ивент создателя")
		assert.Contains(t, foundIDs, event2ID, "Список должен содержать ивент участника")
		assert.NotContains(t, foundIDs, event3ID, "Список НЕ должен содержать абсолютно чужой ивент")
	})
	t.Run("GetParticipant - success", func(t *testing.T) {
		userID := uuid.New()
		eventID := uuid.New()

		require.NoError(t, repo.CreateEvent(ctx, models.Events{
			ID: eventID, CreatorID: uuid.New(), Title: "Party", EventCode: "PARTY1",
			Status: models.StatusDraft, StartsAt: time.Now().Add(time.Hour).Truncate(time.Second),
		}))

		_, joined, err := repo.JoinEvent(ctx, userID, eventID, false)
		require.NoError(t, err)
		require.True(t, joined)

		p, err := repo.GetParticipant(ctx, userID, eventID)

		assert.NoError(t, err)
		assert.Equal(t, userID, p.UserID)
		assert.Equal(t, eventID, p.EventID)
		assert.False(t, p.IsOwner, "Должен быть обычным участником")
	})

	t.Run("GetParticipant - not found", func(t *testing.T) {
		_, err := repo.GetParticipant(ctx, uuid.New(), uuid.New())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "participant not found")
	})
}
