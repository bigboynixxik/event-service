package services_test

import (
	"context"
	"eventify-events/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MockEventRepository struct {
	Events       map[uuid.UUID]models.Events
	Participants map[uuid.UUID][]models.EventParticipants
	Checklist    map[uuid.UUID][]models.ChecklistItems
	Invites      map[string]models.EventInvites
	EventCodes   map[string]uuid.UUID
}

func NewMockRepo() *MockEventRepository {
	return &MockEventRepository{
		Events:       make(map[uuid.UUID]models.Events),
		Participants: make(map[uuid.UUID][]models.EventParticipants),
		Checklist:    make(map[uuid.UUID][]models.ChecklistItems),
		Invites:      make(map[string]models.EventInvites),
		EventCodes:   make(map[string]uuid.UUID),
	}
}

func (m *MockEventRepository) CreateEvent(_ context.Context, e models.Events) error {
	m.Events[e.ID] = e
	if e.EventCode != "" {
		m.EventCodes[e.EventCode] = e.ID
	}
	return nil
}

func (m *MockEventRepository) GetEvent(_ context.Context, id uuid.UUID) (models.Events, error) {
	e, ok := m.Events[id]
	if !ok {
		return models.Events{}, fmt.Errorf("event not found")
	}
	return e, nil
}

func (m *MockEventRepository) ListEvents(_ context.Context) ([]models.Events, error) {
	result := make([]models.Events, 0, len(m.Events))
	for _, e := range m.Events {
		result = append(result, e)
	}
	return result, nil
}

func (m *MockEventRepository) ListUserEvents(_ context.Context, userId uuid.UUID) ([]models.Events, error) {
	result := make([]models.Events, 0)
	for _, e := range m.Events {
		if e.CreatorID == userId {
			result = append(result, e)
			continue
		}
		for _, p := range m.Participants[e.ID] {
			if p.UserID == userId {
				result = append(result, e)
				break
			}
		}
	}
	return result, nil
}

func (m *MockEventRepository) UpdateEvent(_ context.Context, params models.UpdateEventParams, id uuid.UUID) (models.Events, error) {
	e, ok := m.Events[id]
	if !ok {
		return models.Events{}, fmt.Errorf("event not found")
	}
	if params.Title != nil {
		e.Title = *params.Title
	}
	if params.Description != nil {
		e.Description = params.Description
	}
	m.Events[id] = e
	return e, nil
}

func (m *MockEventRepository) GetEventByCode(_ context.Context, code string) (models.Events, error) {
	id, ok := m.EventCodes[code]
	if !ok {
		return models.Events{}, fmt.Errorf("event with code %s not found", code)
	}
	return m.Events[id], nil
}

func (m *MockEventRepository) JoinEvent(_ context.Context, userId uuid.UUID, eventId uuid.UUID, isOwner bool) (uuid.UUID, bool, error) {
	for _, p := range m.Participants[eventId] {
		if p.UserID == userId {
			return eventId, false, nil
		}
	}
	m.Participants[eventId] = append(m.Participants[eventId], models.EventParticipants{
		ID:      uuid.New(),
		UserID:  userId,
		EventID: eventId,
		IsOwner: isOwner,
		Status:  "confirmed",
	})
	return eventId, true, nil
}

func (m *MockEventRepository) RemoveParticipant(_ context.Context, participantId uuid.UUID, eventId uuid.UUID) (bool, error) {
	participants := m.Participants[eventId]
	for i, p := range participants {
		if p.UserID == participantId && !p.IsOwner {
			m.Participants[eventId] = append(participants[:i], participants[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *MockEventRepository) GetEventParticipants(_ context.Context, eventId uuid.UUID) ([]models.EventParticipants, error) {
	return m.Participants[eventId], nil
}

func (m *MockEventRepository) GetParticipant(_ context.Context, userId uuid.UUID, eventId uuid.UUID) (models.EventParticipants, error) {
	for _, p := range m.Participants[eventId] {
		if p.UserID == userId {
			return p, nil
		}
	}
	return models.EventParticipants{}, fmt.Errorf("participant not found")
}

func (m *MockEventRepository) CancelEvent(_ context.Context, eventId uuid.UUID) (bool, error) {
	e, ok := m.Events[eventId]
	if !ok {
		return false, nil
	}
	e.Status = models.StatusCancelled
	m.Events[eventId] = e
	return true, nil
}

func (m *MockEventRepository) CreateInviteLink(_ context.Context, eventId uuid.UUID, inviteType string, expiresAt *time.Time) (string, error) {
	e, ok := m.Events[eventId]
	if !ok {
		return "", fmt.Errorf("event not found")
	}
	token := uuid.New().String()[:16]
	var maxUses *int
	if inviteType == "single" {
		v := 1
		maxUses = &v
	}
	m.Invites[token] = models.EventInvites{
		ID:         uuid.New(),
		EventID:    eventId,
		Token:      token,
		InviteType: models.EventInvitesType(inviteType),
		MaxUses:    maxUses,
		ExpiresAt:  expiresAt,
	}
	return e.EventCode, nil
}

func (m *MockEventRepository) AddChecklistItem(_ context.Context, e models.ChecklistItems) (uuid.UUID, error) {
	m.Checklist[e.EventID] = append(m.Checklist[e.EventID], e)
	return e.ID, nil
}

func (m *MockEventRepository) GetEventChecklist(_ context.Context, eventId uuid.UUID) ([]models.ChecklistItems, error) {
	return m.Checklist[eventId], nil
}

func (m *MockEventRepository) RemoveChecklistItem(_ context.Context, itemId uuid.UUID, eventId uuid.UUID) (bool, error) {
	items := m.Checklist[eventId]
	for i, item := range items {
		if item.ID == itemId {
			m.Checklist[eventId] = append(items[:i], items[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *MockEventRepository) MarkItemPurchased(_ context.Context, eventId uuid.UUID, itemId uuid.UUID, buyerId *uuid.UUID, isPurchased *bool) (bool, error) {
	items := m.Checklist[eventId]
	for i, item := range items {
		if item.ID == itemId {
			if isPurchased != nil {
				items[i].IsPurchased = *isPurchased
			}
			m.Checklist[eventId] = items
			return true, nil
		}
	}
	return false, nil
}

func (m *MockEventRepository) GetInviteByToken(_ context.Context, token string) (models.EventInvites, error) {
	invite, ok := m.Invites[token]
	if !ok {
		return models.EventInvites{}, fmt.Errorf("invite not found")
	}
	return invite, nil
}

func (m *MockEventRepository) UseInvite(_ context.Context, inviteId uuid.UUID) (bool, error) {
	for token, invite := range m.Invites {
		if invite.ID == inviteId {
			if invite.MaxUses != nil && invite.UsedCount >= *invite.MaxUses {
				return false, nil
			}
			invite.UsedCount++
			m.Invites[token] = invite
			return true, nil
		}
	}
	return false, fmt.Errorf("invite not found")
}