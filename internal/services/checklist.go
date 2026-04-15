package services

import (
	"context"
	"eventify-events/internal/models"
	"eventify-events/internal/repository"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ChecklistService struct {
	repo repository.EventRepository
}

func NewChecklistService(repo repository.EventRepository) *ChecklistService {
	return &ChecklistService{repo: repo}
}

func (s *ChecklistService) AddChecklistItem(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, title string, quantity int, unit string) (uuid.UUID, error) {
	_, err := s.repo.GetParticipant(ctx, callerID, eventID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("Service.AddChecklistItem participant check: %w", err)
	}

	event, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("Service.AddChecklistItem: %w", err)
	}
	err = checkEventStatus(event.Status)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("Service.AddChecklistItem: %w", err)
	}
	item := models.ChecklistItems{
		ID:        uuid.New(),
		EventID:   eventID,
		Title:     title,
		Quantity:  quantity,
		IsPurchased: false,
		Unit:      &unit,
		CreatedAt: time.Now(),
	}

	itemId, err := s.repo.AddChecklistItem(ctx, item)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("Service.AddChecklistItem: %w", err)
	}

	return itemId, nil
}

func (s *ChecklistService) RemoveChecklistItem(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, itemID uuid.UUID) (bool, error){


	err := checkPermission(ctx, s.repo, callerID, eventID, "can_manage_checklist")
	if err != nil {
		return false, fmt.Errorf("Service.RemoveChecklistItem: %w", err)
	}
	event, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("Service.RemoveChecklistItem: %w", err)
	}
	err = checkEventStatus(event.Status)
	if err != nil {
		return false, fmt.Errorf("Service.RemoveChecklistItem: %w", err)
	}

	return s.repo.RemoveChecklistItem(ctx, itemID, eventID)
	
}

func (s *ChecklistService) MarkItemPurchased(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID, itemID uuid.UUID, buyerID *uuid.UUID, isPurchased *bool) (bool, error) {

	_, err := s.repo.GetParticipant(ctx, callerID, eventID)
	if err != nil {
		return false, fmt.Errorf("Service.MarkItemPurchased: %w", err)
	}

	event, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("Service.MarkItemPurchased: %w", err)
	}
	err = checkEventStatus(event.Status)
	if err != nil {
		return false, fmt.Errorf("Service.MarkItemPurchased: %w", err)
	}
	return s.repo.MarkItemPurchased(ctx, eventID, itemID, buyerID, isPurchased)
}

func (s *ChecklistService) GetEventChecklist(ctx context.Context, callerID uuid.UUID, eventID uuid.UUID) ([]models.ChecklistItems, error) {

	event, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return []models.ChecklistItems{}, fmt.Errorf("Service.GetEventChecklist: %w", err)
	}
	if event.IsPrivate {
		_, err := s.repo.GetParticipant(ctx, callerID, eventID)
		if err != nil {
			return []models.ChecklistItems{}, fmt.Errorf("Service.GetEventChecklist: %w", err)
		}
	}
	return s.repo.GetEventChecklist(ctx, eventID)
}
