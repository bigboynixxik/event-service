package services_test

import (
	"context"
	"eventify-events/internal/models"
	"eventify-events/internal/services"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func setupEventService() (*services.EventService, *MockEventRepository) {
	repo := NewMockRepo()
	svc := services.NewEventService(repo)
	return svc, repo
}

func createTestEvent(repo *MockEventRepository, overrides ...func(*models.Events)) models.Events {
	e := models.Events{
		ID:        uuid.New(),
		CreatorID: uuid.New(),
		Title:     "Test Event",
		Status:    models.StatusActive,
		IsPrivate: false,
		StartsAt:  time.Now().Add(time.Hour),
		EventCode: uuid.New().String()[:8],
		Duration:  pgtype.Interval{Microseconds: 120 * 60 * 1000000, Valid: true},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, fn := range overrides {
		fn(&e)
	}
	repo.Events[e.ID] = e
	repo.EventCodes[e.EventCode] = e.ID
	return e
}

func addOwner(repo *MockEventRepository, userID uuid.UUID, eventID uuid.UUID) {
	repo.Participants[eventID] = append(repo.Participants[eventID], models.EventParticipants{
		ID:                    uuid.New(),
		UserID:                userID,
		EventID:               eventID,
		IsOwner:               true,
		CanEditEvent:          true,
		CanManageParticipants: true,
		CanManageChecklist:    true,
		Status:                "confirmed",
	})
}

func addParticipant(repo *MockEventRepository, userID uuid.UUID, eventID uuid.UUID) {
	repo.Participants[eventID] = append(repo.Participants[eventID], models.EventParticipants{
		ID:      uuid.New(),
		UserID:  userID,
		EventID: eventID,
		IsOwner: false,
		Status:  "confirmed",
	})
}

// === GetEvent ===

func TestGetEvent_Success(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)

	result, err := svc.GetEvent(context.Background(), event.ID)

	assert.NoError(t, err)
	assert.Equal(t, event.ID, result.ID)
	assert.Equal(t, event.Title, result.Title)
}

func TestGetEvent_NotFound(t *testing.T) {
	svc, _ := setupEventService()

	_, err := svc.GetEvent(context.Background(), uuid.New())

	assert.Error(t, err)
}

// === ListEvents ===

func TestListEvents_FiltersPrivateAndCancelled(t *testing.T) {
	svc, repo := setupEventService()

	createTestEvent(repo, func(e *models.Events) { e.Title = "Public Active" })
	createTestEvent(repo, func(e *models.Events) { e.Title = "Private"; e.IsPrivate = true })
	createTestEvent(repo, func(e *models.Events) { e.Title = "Cancelled"; e.Status = models.StatusCancelled })

	result, err := svc.ListEvents(context.Background(), services.ListEventsFilter{})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Public Active", result[0].Title)
}

func TestListEvents_FilterByTitle(t *testing.T) {
	svc, repo := setupEventService()

	createTestEvent(repo, func(e *models.Events) { e.Title = "День рождения Маши" })
	createTestEvent(repo, func(e *models.Events) { e.Title = "Пикник в парке" })

	result, err := svc.ListEvents(context.Background(), services.ListEventsFilter{
		Title: ptr("рождения"),
	})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result[0].Title, "рождения")
}

func TestListEvents_FilterByTitleCaseInsensitive(t *testing.T) {
	svc, repo := setupEventService()

	createTestEvent(repo, func(e *models.Events) { e.Title = "День Рождения" })

	result, err := svc.ListEvents(context.Background(), services.ListEventsFilter{
		Title: ptr("день рождения"),
	})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListEvents_FilterByDate(t *testing.T) {
	svc, repo := setupEventService()

	now := time.Now()
	createTestEvent(repo, func(e *models.Events) {
		e.Title = "Tomorrow"
		e.StartsAt = now.Add(24 * time.Hour)
	})
	createTestEvent(repo, func(e *models.Events) {
		e.Title = "Next Week"
		e.StartsAt = now.Add(7 * 24 * time.Hour)
	})

	after := now.Add(2 * 24 * time.Hour)
	result, err := svc.ListEvents(context.Background(), services.ListEventsFilter{
		StartsAfter: &after,
	})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Next Week", result[0].Title)
}

func TestListEvents_MultipleFiltersAND(t *testing.T) {
	svc, repo := setupEventService()

	now := time.Now()
	createTestEvent(repo, func(e *models.Events) {
		e.Title = "День рождения"
		e.StartsAt = now.Add(24 * time.Hour)
	})
	createTestEvent(repo, func(e *models.Events) {
		e.Title = "День рождения далёкий"
		e.StartsAt = now.Add(30 * 24 * time.Hour)
	})

	before := now.Add(7 * 24 * time.Hour)
	result, err := svc.ListEvents(context.Background(), services.ListEventsFilter{
		Title:        ptr("рождения"),
		StartsBefore: &before,
	})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "День рождения", result[0].Title)
}

func TestListEvents_NoResults(t *testing.T) {
	svc, repo := setupEventService()

	createTestEvent(repo, func(e *models.Events) { e.Title = "Пикник" })

	result, err := svc.ListEvents(context.Background(), services.ListEventsFilter{
		Title: ptr("несуществующий"),
	})

	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

// === CreateEvent ===

func TestCreateEvent_Success(t *testing.T) {
	svc, repo := setupEventService()
	callerID := uuid.New()

	params := services.EventInputParams{
		Title:     "New Party",
		IsPrivate: false,
		StartsAt:  time.Now().Add(time.Hour),
		Duration:  pgtype.Interval{Microseconds: 60 * 60 * 1000000, Valid: true},
	}

	event, err := svc.CreateEvent(context.Background(), callerID, params)

	assert.NoError(t, err)
	assert.Equal(t, "New Party", event.Title)
	assert.Equal(t, models.StatusDraft, event.Status)
	assert.Equal(t, callerID, event.CreatorID)

	// Owner должен быть добавлен как участник
	participants := repo.Participants[event.ID]
	require.Len(t, participants, 1)
	assert.Equal(t, callerID, participants[0].UserID)
	assert.True(t, participants[0].IsOwner)
}

// === JoinEvent ===

func TestJoinEvent_Success(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	userID := uuid.New()

	joined, err := svc.JoinEvent(context.Background(), userID, event.EventCode)

	assert.NoError(t, err)
	assert.True(t, joined)
}

func TestJoinEvent_EventFull(t *testing.T) {
	svc, repo := setupEventService()
	maxP := 1
	event := createTestEvent(repo, func(e *models.Events) { e.MaxParticipants = &maxP })
	addParticipant(repo, uuid.New(), event.ID)

	_, err := svc.JoinEvent(context.Background(), uuid.New(), event.EventCode)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "full")
}

func TestJoinEvent_CancelledEvent(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo, func(e *models.Events) { e.Status = models.StatusCancelled })

	_, err := svc.JoinEvent(context.Background(), uuid.New(), event.EventCode)

	assert.Error(t, err)
}

func TestJoinEvent_InvalidCode(t *testing.T) {
	svc, _ := setupEventService()

	_, err := svc.JoinEvent(context.Background(), uuid.New(), "WRONG-CODE")

	assert.Error(t, err)
}

// === LeaveEvent ===

func TestLeaveEvent_Success(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	success, err := svc.LeaveEvent(context.Background(), userID, event.ID)

	assert.NoError(t, err)
	assert.True(t, success)
}

func TestLeaveEvent_OwnerCantLeave(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	_, err := svc.LeaveEvent(context.Background(), ownerID, event.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner")
}

func TestLeaveEvent_NotParticipant(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)

	_, err := svc.LeaveEvent(context.Background(), uuid.New(), event.ID)

	assert.Error(t, err)
}

// === RemoveParticipant ===

func TestRemoveParticipant_Success(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)
	participantID := uuid.New()
	addParticipant(repo, participantID, event.ID)

	success, err := svc.RemoveParticipant(context.Background(), ownerID, participantID, event.ID)

	assert.NoError(t, err)
	assert.True(t, success)
}

func TestRemoveParticipant_CantRemoveOwner(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	_, err := svc.RemoveParticipant(context.Background(), ownerID, ownerID, event.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creator")
}

func TestRemoveParticipant_NoPermission(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	callerID := uuid.New()
	addParticipant(repo, callerID, event.ID)
	targetID := uuid.New()
	addParticipant(repo, targetID, event.ID)

	_, err := svc.RemoveParticipant(context.Background(), callerID, targetID, event.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}

// === UpdateEvent ===

func TestUpdateEvent_Success(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	newTitle := "Updated Title"
	result, err := svc.UpdateEvent(context.Background(), ownerID, event.ID, models.UpdateEventParams{
		Title: &newTitle,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", result.Title)
}

func TestUpdateEvent_CancelledEvent(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo, func(e *models.Events) { e.Status = models.StatusCancelled })
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	_, err := svc.UpdateEvent(context.Background(), ownerID, event.ID, models.UpdateEventParams{
		Title: ptr("New"),
	})

	assert.Error(t, err)
}

// === CancelEvent ===

func TestCancelEvent_Success(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	success, err := svc.CancelEvent(context.Background(), ownerID, event.ID)

	assert.NoError(t, err)
	assert.True(t, success)
	assert.Equal(t, models.StatusCancelled, repo.Events[event.ID].Status)
}

func TestCancelEvent_AlreadyCancelled(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo, func(e *models.Events) { e.Status = models.StatusCancelled })
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	_, err := svc.CancelEvent(context.Background(), ownerID, event.ID)

	assert.Error(t, err)
}

func TestCancelEvent_NoPermission(t *testing.T) {
	svc, repo := setupEventService()
	event := createTestEvent(repo)
	randomUser := uuid.New()
	addParticipant(repo, randomUser, event.ID)

	_, err := svc.CancelEvent(context.Background(), randomUser, event.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}