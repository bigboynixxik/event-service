package services_test

import (
	"context"
	"eventify-events/internal/models"
	"eventify-events/internal/services"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupChecklistService() (*services.ChecklistService, *MockEventRepository) {
	repo := NewMockRepo()
	svc := services.NewChecklistService(repo)
	return svc, repo
}

// === AddChecklistItem ===

func TestAddChecklistItem_Success(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo)
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	itemID, err := svc.AddChecklistItem(context.Background(), userID, event.ID, "Салфетки", 2, "уп.")

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, itemID)
	assert.Len(t, repo.Checklist[event.ID], 1)
	assert.Equal(t, "Салфетки", repo.Checklist[event.ID][0].Title)
}

func TestAddChecklistItem_NotParticipant(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo)

	_, err := svc.AddChecklistItem(context.Background(), uuid.New(), event.ID, "Item", 1, "шт.")

	assert.Error(t, err)
}

func TestAddChecklistItem_CancelledEvent(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo, func(e *models.Events) { e.Status = models.StatusCancelled })
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	_, err := svc.AddChecklistItem(context.Background(), userID, event.ID, "Item", 1, "шт.")

	assert.Error(t, err)
}

// === RemoveChecklistItem ===

func TestRemoveChecklistItem_Success(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo)
	ownerID := uuid.New()
	addOwner(repo, ownerID, event.ID)

	itemID := uuid.New()
	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: itemID, EventID: event.ID, Title: "Water"},
	}

	success, err := svc.RemoveChecklistItem(context.Background(), ownerID, event.ID, itemID)

	assert.NoError(t, err)
	assert.True(t, success)
	assert.Len(t, repo.Checklist[event.ID], 0)
}

func TestRemoveChecklistItem_NoPermission(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo)
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	itemID := uuid.New()
	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: itemID, EventID: event.ID, Title: "Water"},
	}

	_, err := svc.RemoveChecklistItem(context.Background(), userID, event.ID, itemID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}

// === MarkItemPurchased ===

func TestMarkItemPurchased_Success(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo)
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	itemID := uuid.New()
	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: itemID, EventID: event.ID, Title: "Milk", IsPurchased: false},
	}

	isPurchased := true
	success, err := svc.MarkItemPurchased(context.Background(), userID, event.ID, itemID, &userID, &isPurchased)

	assert.NoError(t, err)
	assert.True(t, success)
	assert.True(t, repo.Checklist[event.ID][0].IsPurchased)
}

func TestMarkItemPurchased_NotParticipant(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo)

	itemID := uuid.New()
	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: itemID, EventID: event.ID, Title: "Bread"},
	}

	_, err := svc.MarkItemPurchased(context.Background(), uuid.New(), event.ID, itemID, nil, nil)

	assert.Error(t, err)
}

func TestMarkItemPurchased_CancelledEvent(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo, func(e *models.Events) { e.Status = models.StatusCancelled })
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	itemID := uuid.New()
	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: itemID, EventID: event.ID, Title: "Eggs"},
	}

	_, err := svc.MarkItemPurchased(context.Background(), userID, event.ID, itemID, nil, nil)

	assert.Error(t, err)
}

// === GetEventChecklist ===

func TestGetEventChecklist_PublicEvent(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo, func(e *models.Events) { e.IsPrivate = false })

	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: uuid.New(), EventID: event.ID, Title: "Item 1"},
		{ID: uuid.New(), EventID: event.ID, Title: "Item 2"},
	}

	// Любой пользователь может смотреть чеклист публичного ивента
	result, err := svc.GetEventChecklist(context.Background(), uuid.New(), event.ID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetEventChecklist_PrivateEvent_Participant(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo, func(e *models.Events) { e.IsPrivate = true })
	userID := uuid.New()
	addParticipant(repo, userID, event.ID)

	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: uuid.New(), EventID: event.ID, Title: "Secret Item"},
	}

	result, err := svc.GetEventChecklist(context.Background(), userID, event.ID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestGetEventChecklist_PrivateEvent_NotParticipant(t *testing.T) {
	svc, repo := setupChecklistService()
	event := createTestEvent(repo, func(e *models.Events) { e.IsPrivate = true })

	repo.Checklist[event.ID] = []models.ChecklistItems{
		{ID: uuid.New(), EventID: event.ID, Title: "Secret Item"},
	}

	_, err := svc.GetEventChecklist(context.Background(), uuid.New(), event.ID)

	assert.Error(t, err)
}