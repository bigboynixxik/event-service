package api_test

import (
	"context"
	"eventify-events/internal/api"
	"eventify-events/internal/models"
	"eventify-events/internal/services"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	v1 "eventify-events/pkg/api/v1"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// === Mock Event Service ===

type MockEventService struct {
	Events       map[uuid.UUID]*models.Events
	Participants map[uuid.UUID][]models.EventParticipants
}

func NewMockEventService() *MockEventService {
	return &MockEventService{
		Events:       make(map[uuid.UUID]*models.Events),
		Participants: make(map[uuid.UUID][]models.EventParticipants),
	}
}

func (m *MockEventService) GetEvent(_ context.Context, id uuid.UUID) (*models.Events, error) {
	e, ok := m.Events[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return e, nil
}

func (m *MockEventService) ListEvents(_ context.Context, filter services.ListEventsFilter) ([]models.Events, error) {
	result := make([]models.Events, 0)
	for _, e := range m.Events {
		result = append(result, *e)
	}
	return result, nil
}

func (m *MockEventService) ListUserEvents(_ context.Context, userID uuid.UUID) ([]models.Events, error) {
	result := make([]models.Events, 0)
	for _, e := range m.Events {
		if e.CreatorID == userID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *MockEventService) CreateEvent(_ context.Context, callerID uuid.UUID, params services.EventInputParams) (models.Events, error) {
	e := models.Events{
		ID:        uuid.New(),
		CreatorID: callerID,
		Title:     params.Title,
		Status:    models.StatusDraft,
		StartsAt:  params.StartsAt,
		Duration:  params.Duration,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.Events[e.ID] = &e
	return e, nil
}

func (m *MockEventService) UpdateEvent(_ context.Context, callerID uuid.UUID, eventID uuid.UUID, params models.UpdateEventParams) (models.Events, error) {
	e, ok := m.Events[eventID]
	if !ok {
		return models.Events{}, fmt.Errorf("not found")
	}
	if params.Title != nil {
		e.Title = *params.Title
	}
	return *e, nil
}

func (m *MockEventService) JoinEvent(_ context.Context, userID uuid.UUID, code string) (bool, error) {
	if code == "" {
		return false, fmt.Errorf("empty code")
	}
	return true, nil
}

func (m *MockEventService) LeaveEvent(_ context.Context, callerID uuid.UUID, eventID uuid.UUID) (bool, error) {
	_, ok := m.Events[eventID]
	if !ok {
		return false, fmt.Errorf("not found")
	}
	return true, nil
}

func (m *MockEventService) RemoveParticipant(_ context.Context, callerID uuid.UUID, participantID uuid.UUID, eventID uuid.UUID) (bool, error) {
	return true, nil
}

func (m *MockEventService) GetEventParticipants(_ context.Context, eventID uuid.UUID) ([]models.EventParticipants, error) {
	return m.Participants[eventID], nil
}

func (m *MockEventService) CancelEvent(_ context.Context, callerID uuid.UUID, eventID uuid.UUID) (bool, error) {
	e, ok := m.Events[eventID]
	if !ok {
		return false, fmt.Errorf("not found")
	}
	e.Status = models.StatusCancelled
	return true, nil
}

func (m *MockEventService) CreateInviteLink(_ context.Context, callerID uuid.UUID, eventID uuid.UUID, inviteType string, expiresAt *time.Time) (string, error) {
	return "invite-code-123", nil
}

// === Mock Checklist Service ===

type MockChecklistService struct {
	Items map[uuid.UUID][]models.ChecklistItems
}

func NewMockChecklistService() *MockChecklistService {
	return &MockChecklistService{
		Items: make(map[uuid.UUID][]models.ChecklistItems),
	}
}

func (m *MockChecklistService) AddChecklistItem(_ context.Context, callerID uuid.UUID, eventID uuid.UUID, title string, quantity int, unit string) (uuid.UUID, error) {
	id := uuid.New()
	u := unit
	m.Items[eventID] = append(m.Items[eventID], models.ChecklistItems{
		ID:       id,
		EventID:  eventID,
		Title:    title,
		Quantity: quantity,
		Unit:     &u,
	})
	return id, nil
}

func (m *MockChecklistService) RemoveChecklistItem(_ context.Context, callerID uuid.UUID, eventID uuid.UUID, itemID uuid.UUID) (bool, error) {
	return true, nil
}

func (m *MockChecklistService) MarkItemPurchased(_ context.Context, callerID uuid.UUID, eventID uuid.UUID, itemID uuid.UUID, buyerID *uuid.UUID, isPurchased *bool) (bool, error) {
	return true, nil
}

func (m *MockChecklistService) GetEventChecklist(_ context.Context, callerID uuid.UUID, eventID uuid.UUID) ([]models.ChecklistItems, error) {
	return m.Items[eventID], nil
}

// === Helpers ===

func newHandler() *api.EventHandler {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return api.NewEventHandler(NewMockEventService(), NewMockChecklistService(), log)
}

func newHandlerWithServices() (*api.EventHandler, *MockEventService, *MockChecklistService) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	es := NewMockEventService()
	cs := NewMockChecklistService()
	return api.NewEventHandler(es, cs, log), es, cs
}

func ctxWithUser(userID uuid.UUID) context.Context {
	return api.ContextWithUserID(context.Background(), userID)
}

func assertGRPCCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error")
	assert.Equal(t, expected, st.Code())
}

// === GetEvent Tests ===

func TestHandler_GetEvent_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	eventID := uuid.New()
	es.Events[eventID] = &models.Events{
		ID:       eventID,
		Title:    "Test",
		Status:   models.StatusActive,
		StartsAt: time.Now(),
		Duration: pgtype.Interval{Microseconds: 60 * 60 * 1000000, Valid: true},
	}

	resp, err := h.GetEvent(context.Background(), &v1.GetEventRequest{Id: eventID.String()})

	assert.NoError(t, err)
	assert.Equal(t, eventID.String(), resp.Event.Id)
	assert.Equal(t, "Test", resp.Event.Title)
}

func TestHandler_GetEvent_InvalidID(t *testing.T) {
	h := newHandler()

	_, err := h.GetEvent(context.Background(), &v1.GetEventRequest{Id: "not-a-uuid"})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestHandler_GetEvent_NotFound(t *testing.T) {
	h := newHandler()

	_, err := h.GetEvent(context.Background(), &v1.GetEventRequest{Id: uuid.New().String()})

	assertGRPCCode(t, err, codes.NotFound)
}

// === ListEvents Tests ===

func TestHandler_ListEvents_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	id := uuid.New()
	es.Events[id] = &models.Events{
		ID:       id,
		Title:    "Party",
		Status:   models.StatusActive,
		StartsAt: time.Now(),
		Duration: pgtype.Interval{Microseconds: 60 * 60 * 1000000, Valid: true},
	}

	resp, err := h.ListEvents(context.Background(), &v1.ListEventsRequest{})

	assert.NoError(t, err)
	assert.Len(t, resp.Events, 1)
}

func TestHandler_ListEvents_WithFilters(t *testing.T) {
	h := newHandler()
	title := "birthday"

	resp, err := h.ListEvents(context.Background(), &v1.ListEventsRequest{
		Title: &title,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

// === ListUserEvents Tests ===

func TestHandler_ListUserEvents_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	userID := uuid.New()
	id := uuid.New()
	es.Events[id] = &models.Events{
		ID:        id,
		CreatorID: userID,
		Title:     "My Event",
		Status:    models.StatusActive,
		StartsAt:  time.Now(),
		Duration:  pgtype.Interval{Microseconds: 60 * 60 * 1000000, Valid: true},
	}

	resp, err := h.ListUserEvents(ctxWithUser(userID), &v1.ListUserEventsRequest{})

	assert.NoError(t, err)
	assert.Len(t, resp.Events, 1)
}

func TestHandler_ListUserEvents_Unauthenticated(t *testing.T) {
	h := newHandler()

	_, err := h.ListUserEvents(context.Background(), &v1.ListUserEventsRequest{})

	assertGRPCCode(t, err, codes.Unauthenticated)
}

// === CreateEvent Tests ===

func TestHandler_CreateEvent_Success(t *testing.T) {
	h := newHandler()
	userID := uuid.New()

	resp, err := h.CreateEvent(ctxWithUser(userID), &v1.CreateEventRequest{
		Title:           "New Party",
		Description:     "Fun event",
		IsPrivate:       false,
		DurationMinutes: 120,
		StartsAt:        timestamppb.New(time.Now().Add(time.Hour)),
		LocationName:    "Park",
		MaxParticipants: 20,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
}

func TestHandler_CreateEvent_Unauthenticated(t *testing.T) {
	h := newHandler()

	_, err := h.CreateEvent(context.Background(), &v1.CreateEventRequest{
		Title: "Party",
	})

	assertGRPCCode(t, err, codes.Unauthenticated)
}

func TestHandler_CreateEvent_WithCoords(t *testing.T) {
	h := newHandler()
	userID := uuid.New()
	coords := "55.7558,37.6173"

	resp, err := h.CreateEvent(ctxWithUser(userID), &v1.CreateEventRequest{
		Title:          "Geo Party",
		DurationMinutes: 60,
		StartsAt:       timestamppb.New(time.Now().Add(time.Hour)),
		LocationCoords: &coords,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
}

func TestHandler_CreateEvent_InvalidCoords(t *testing.T) {
	h := newHandler()
	userID := uuid.New()
	coords := "not,valid,coords"

	_, err := h.CreateEvent(ctxWithUser(userID), &v1.CreateEventRequest{
		Title:          "Bad Coords",
		DurationMinutes: 60,
		StartsAt:       timestamppb.New(time.Now().Add(time.Hour)),
		LocationCoords: &coords,
	})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

// === JoinEvent Tests ===

func TestHandler_JoinEvent_Success(t *testing.T) {
	h := newHandler()
	userID := uuid.New()

	resp, err := h.JoinEvent(ctxWithUser(userID), &v1.JoinEventRequest{
		EventCode: "valid-code",
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_JoinEvent_Unauthenticated(t *testing.T) {
	h := newHandler()

	_, err := h.JoinEvent(context.Background(), &v1.JoinEventRequest{
		EventCode: "code",
	})

	assertGRPCCode(t, err, codes.Unauthenticated)
}

// === LeaveEvent Tests ===

func TestHandler_LeaveEvent_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	userID := uuid.New()
	eventID := uuid.New()
	es.Events[eventID] = &models.Events{ID: eventID, Status: models.StatusActive}

	resp, err := h.LeaveEvent(ctxWithUser(userID), &v1.LeaveEventRequest{
		EventId: eventID.String(),
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_LeaveEvent_InvalidEventID(t *testing.T) {
	h := newHandler()

	_, err := h.LeaveEvent(ctxWithUser(uuid.New()), &v1.LeaveEventRequest{
		EventId: "bad-uuid",
	})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

// === CancelEvent Tests ===

func TestHandler_CancelEvent_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	userID := uuid.New()
	eventID := uuid.New()
	es.Events[eventID] = &models.Events{ID: eventID, Status: models.StatusActive}

	resp, err := h.CancelEvent(ctxWithUser(userID), &v1.CancelEventRequest{
		EventId: eventID.String(),
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

// === RemoveParticipant Tests ===

func TestHandler_RemoveParticipant_Success(t *testing.T) {
	h := newHandler()
	callerID := uuid.New()
	participantID := uuid.New()
	eventID := uuid.New()

	resp, err := h.RemoveParticipant(ctxWithUser(callerID), &v1.RemoveParticipantRequest{
		EventId:       eventID.String(),
		ParticipantId: participantID.String(),
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_RemoveParticipant_InvalidParticipantID(t *testing.T) {
	h := newHandler()

	_, err := h.RemoveParticipant(ctxWithUser(uuid.New()), &v1.RemoveParticipantRequest{
		EventId:       uuid.New().String(),
		ParticipantId: "bad-id",
	})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

// === GetEventParticipants Tests ===

func TestHandler_GetEventParticipants_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	eventID := uuid.New()
	userID := uuid.New()
	es.Participants[eventID] = []models.EventParticipants{
		{ID: uuid.New(), UserID: userID, EventID: eventID, Status: "confirmed"},
	}

	resp, err := h.GetEventParticipants(context.Background(), &v1.GetEventParticipantsRequest{
		EventId: eventID.String(),
	})

	assert.NoError(t, err)
	assert.Len(t, resp.Participants, 1)
	assert.Equal(t, userID.String(), resp.Participants[0].ParticipantId)
}

// === CreateInviteLink Tests ===

func TestHandler_CreateInviteLink_Success(t *testing.T) {
	h := newHandler()
	userID := uuid.New()

	resp, err := h.CreateInviteLink(ctxWithUser(userID), &v1.CreateInviteLinkRequest{
		EventId:    uuid.New().String(),
		InviteType: "single",
		ExpiresAt:  timestamppb.New(time.Now().Add(24 * time.Hour)),
	})

	assert.NoError(t, err)
	assert.Equal(t, "invite-code-123", resp.EventCode)
}

// === AddChecklistItem Tests ===

func TestHandler_AddChecklistItem_Success(t *testing.T) {
	h := newHandler()
	userID := uuid.New()

	resp, err := h.AddChecklistItem(ctxWithUser(userID), &v1.AddChecklistItemRequest{
		EventId:  uuid.New().String(),
		Title:    "Napkins",
		Quantity: 3,
		Unit:     "packs",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.ItemId)
}

func TestHandler_AddChecklistItem_Unauthenticated(t *testing.T) {
	h := newHandler()

	_, err := h.AddChecklistItem(context.Background(), &v1.AddChecklistItemRequest{
		EventId: uuid.New().String(),
		Title:   "Item",
	})

	assertGRPCCode(t, err, codes.Unauthenticated)
}

// === RemoveChecklistItem Tests ===

func TestHandler_RemoveChecklistItem_Success(t *testing.T) {
	h := newHandler()

	resp, err := h.RemoveChecklistItem(ctxWithUser(uuid.New()), &v1.RemoveChecklistItemRequest{
		EventId: uuid.New().String(),
		ItemId:  uuid.New().String(),
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_RemoveChecklistItem_InvalidItemID(t *testing.T) {
	h := newHandler()

	_, err := h.RemoveChecklistItem(ctxWithUser(uuid.New()), &v1.RemoveChecklistItemRequest{
		EventId: uuid.New().String(),
		ItemId:  "bad-id",
	})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

// === MarkItemPurchased Tests ===

func TestHandler_MarkItemPurchased_Success(t *testing.T) {
	h := newHandler()
	buyerID := uuid.New().String()

	resp, err := h.MarkItemPurchased(ctxWithUser(uuid.New()), &v1.MarkItemPurchasedRequest{
		EventId:     uuid.New().String(),
		ItemId:      uuid.New().String(),
		BuyerId:     &buyerID,
		IsPurchased: ptr(true),
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_MarkItemPurchased_InvalidBuyerID(t *testing.T) {
	h := newHandler()
	badID := "not-uuid"

	_, err := h.MarkItemPurchased(ctxWithUser(uuid.New()), &v1.MarkItemPurchasedRequest{
		EventId: uuid.New().String(),
		ItemId:  uuid.New().String(),
		BuyerId: &badID,
	})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestHandler_MarkItemPurchased_NoBuyer(t *testing.T) {
	h := newHandler()

	resp, err := h.MarkItemPurchased(ctxWithUser(uuid.New()), &v1.MarkItemPurchasedRequest{
		EventId:     uuid.New().String(),
		ItemId:      uuid.New().String(),
		IsPurchased: ptr(true),
	})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

// === GetEventChecklist Tests ===

func TestHandler_GetEventChecklist_Success(t *testing.T) {
	h, _, cs := newHandlerWithServices()
	eventID := uuid.New()
	u := "packs"
	cs.Items[eventID] = []models.ChecklistItems{
		{ID: uuid.New(), EventID: eventID, Title: "Water", Quantity: 2, Unit: &u},
	}

	resp, err := h.GetEventChecklist(ctxWithUser(uuid.New()), &v1.GetEventChecklistRequest{
		EventId: eventID.String(),
	})

	assert.NoError(t, err)
	assert.Len(t, resp.Checklist, 1)
	assert.Equal(t, "Water", resp.Checklist[0].Title)
}

// === UpdateEvent Tests ===

func TestHandler_UpdateEvent_Success(t *testing.T) {
	h, es, _ := newHandlerWithServices()
	userID := uuid.New()
	eventID := uuid.New()
	es.Events[eventID] = &models.Events{
		ID:       eventID,
		Title:    "Old Title",
		Status:   models.StatusActive,
		StartsAt: time.Now(),
		Duration: pgtype.Interval{Microseconds: 60 * 60 * 1000000, Valid: true},
	}
	newTitle := "New Title"

	resp, err := h.UpdateEvent(ctxWithUser(userID), &v1.UpdateEventRequest{
		EventId: eventID.String(),
		Title:   &newTitle,
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Title", resp.Event.Title)
}

func TestHandler_UpdateEvent_InvalidEventID(t *testing.T) {
	h := newHandler()

	_, err := h.UpdateEvent(ctxWithUser(uuid.New()), &v1.UpdateEventRequest{
		EventId: "bad-id",
	})

	assertGRPCCode(t, err, codes.InvalidArgument)
}

func ptr[T any](v T) *T { return &v }