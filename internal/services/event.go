package services

import (
	"context"
	"eventify-events/internal/models"
	"eventify-events/internal/repository"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type EventService struct {
	repo repository.EventRepository
}

type EventInputParams struct {
	IsPrivate       bool            
	Title           string          
	Description     *string         
	Duration        pgtype.Interval 
	StartsAt        time.Time       
	LocationName    *string         
	MaxParticipants *int            
	LocationCoords  *pgtype.Point   
}

type ListEventsFilter struct {
    Title        *string
    Description  *string
    StartsAfter  *time.Time
    StartsBefore *time.Time
    LocationName *string
}


func NewEventService(repo repository.EventRepository) *EventService {
	return &EventService{repo: repo}
}




func (s *EventService) GetEvent(ctx context.Context, uuid uuid.UUID) (*models.Events, error) {
	event, err := s.repo.GetEvent(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("Service.GetEvent : %w", err)
	}
	return &event, nil
}

func (s *EventService) ListEvents(ctx context.Context, filter ListEventsFilter) ([]models.Events, error) { // Возвращает все неотмененные и публичные ивенты
	events, err := s.repo.ListEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("Service.ListEvents : %w", err)
	}
	filtredEvents := make([]models.Events, 0, len(events))
	for _, e := range events {
		if e.Status != models.StatusCancelled && !e.IsPrivate {
			if filter.Title != nil && !strings.Contains(strings.ToLower(e.Title), strings.ToLower(*filter.Title)) {
				continue

			} 
			if filter.Description != nil && e.Description != nil && !strings.Contains(strings.ToLower(*e.Description), strings.ToLower(*filter.Description)) {
				continue

			} 
			if filter.StartsAfter != nil && !e.StartsAt.After(*filter.StartsAfter) {
				continue

			} 
			if filter.StartsBefore != nil && !e.StartsAt.Before(*filter.StartsBefore) {
				continue

			} 
			if filter.LocationName != nil && e.LocationName != nil && !strings.Contains(strings.ToLower(*e.LocationName), strings.ToLower(*filter.LocationName)) {
				continue
			}
			filtredEvents = append(filtredEvents, e)
		}
	}
	return filtredEvents, nil
}


func (s *EventService) ListUserEvents(ctx context.Context, userId uuid.UUID) ([]models.Events, error) { 
	events, err := s.repo.ListUserEvents(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("Service.ListUserEvents : %w", err)
	}
	return events, nil
}

func (s *EventService) JoinEvent(ctx context.Context, userId uuid.UUID, code string) (bool, error) {
	event, err := s.repo.GetEventByCode(ctx, code)
	if err != nil {
		return false, fmt.Errorf("Service.GetEventByCode : %w", err)
	}
	participants, err := s.repo.GetEventParticipants(ctx, event.ID)
	if err != nil {
		return false, fmt.Errorf("Service.GetEventParticipants : %w", err)
	}


	if event.MaxParticipants != nil && *event.MaxParticipants != 0 && len(participants) >= *event.MaxParticipants {
		return false, fmt.Errorf("event with code %s is full", code)
	}

	// Проверка на статус (не cancelled и не completed)
	if checkEventStatus(event.Status) != nil {
		return false, fmt.Errorf("event with code %s is %s", code, event.Status)
	}
	if event.IsPrivate {
		invite, err := s.repo.GetInviteByToken(ctx, code)
		if err != nil {
			return false, fmt.Errorf("Service.GetInviteByToken : %w", err)
		}
		if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
			return false, fmt.Errorf("invite with code %s is expired", code)
		}
		_, err = s.repo.UseInvite(ctx, invite.ID)
		if err != nil {
			return false, fmt.Errorf("Service.UseInvite : %w", err)
		}
	}

	_, joined, err := s.repo.JoinEvent(ctx, userId, event.ID, false)
	if err != nil {
		return false, fmt.Errorf("Service.JoinEvent : %w", err)
	}
	return joined, nil
}

func (s *EventService) CreateEvent(ctx context.Context, callerId uuid.UUID, eventParams EventInputParams) (models.Events, error) {
	event := models.Events{
		ID: uuid.New(),
		CreatorID: callerId,
		IsPrivate: eventParams.IsPrivate,
		Title: eventParams.Title,
		Description: eventParams.Description,
		StartsAt: eventParams.StartsAt,
		Duration: eventParams.Duration,
		LocationName: eventParams.LocationName,
		LocationCoords: eventParams.LocationCoords,
		MaxParticipants: eventParams.MaxParticipants,
		Status: models.StatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.repo.CreateEvent(ctx, event)
	if err != nil {
		return models.Events{}, fmt.Errorf("Service.CreateEvent : %w", err)
	}
	_, _, err = s.repo.JoinEvent(ctx, callerId, event.ID, true)
	if err != nil {
		return models.Events{}, fmt.Errorf("Service.JoinEvent : %w", err)
	}
	return event, nil
	
}
func (s *EventService) RemoveParticipant(ctx context.Context, callerId uuid.UUID, participantId uuid.UUID, eventId uuid.UUID) (bool, error) {
	err := checkPermission(ctx, s.repo, callerId, eventId, "can_manage_participants")
	if err != nil {
		return false, fmt.Errorf("Service.RemoveParticipant : %w", err)
	}


	participant, err := s.repo.GetParticipant(ctx, participantId, eventId)
	if err != nil {
		return false, fmt.Errorf("Service.RemoveParticipant : %w", err)
	}

	if participant.IsOwner {
		return false, fmt.Errorf("Service.RemoveParticipant : can't remove creator")
	}

	removed, err := s.repo.RemoveParticipant(ctx, participantId, eventId)
	if err != nil {
		return false, fmt.Errorf("Service.RemoveParticipant : %w", err)
	}
	return removed, nil
}


func (s* EventService) GetEventParticipants(ctx context.Context, eventId uuid.UUID) ([]models.EventParticipants, error) {
	participants, err := s.repo.GetEventParticipants(ctx, eventId)
	if err != nil {
		return nil, fmt.Errorf("Service.GetEventParticipants : %w", err)
	}
	return participants, nil
}


func (s* EventService) CancelEvent(ctx context.Context, callerId uuid.UUID, eventId uuid.UUID) (bool, error) {
	err := checkPermission(ctx,s.repo, callerId, eventId, "can_edit_event")
	if err != nil {
		return false, fmt.Errorf("Service.CancelEvent : %w", err)
	}
	event, err := s.repo.GetEvent(ctx, eventId)
	if err != nil {
		return false, fmt.Errorf("Service.CancelEvent : %w", err)
	}

	if checkEventStatus(event.Status) != nil{
		return false, fmt.Errorf("Service.CancelEvent : %s", event.Status)
	}

	return s.repo.CancelEvent(ctx, eventId)
}

func (s* EventService) CreateInviteLink(ctx context.Context, callerId uuid.UUID, eventId uuid.UUID, inviteType string, expiresAt *time.Time) (string, error) { // Можно создать ссылку только для ивентов с доступным статусом
	
	event, err := s.repo.GetEvent(ctx, eventId)
	if err != nil {
		return "", fmt.Errorf("Service.CreateInviteLink : %w", err)
	}

	if checkEventStatus(event.Status) != nil{
		return "", fmt.Errorf("Service.CreateInviteLink : %s", event.Status)
	}
	if event.IsPrivate { // Если приватный - нужно разрешение на создание ссылки
		err := checkPermission(ctx, s.repo, callerId, eventId, "can_manage_participants")
		if err != nil {
			return "", fmt.Errorf("Service.CreateInviteLink : %w", err)
		}
	}
	return s.repo.CreateInviteLink(ctx, eventId, inviteType, expiresAt)
}

func (s* EventService) UpdateEvent(ctx context.Context, callerId uuid.UUID, eventId uuid.UUID, params models.UpdateEventParams) (models.Events, error) {
	err := checkPermission(ctx, s.repo, callerId, eventId, "can_edit_event")
	if err != nil {
		return models.Events{}, fmt.Errorf("Service.UpdateEvent : %w", err)
	}
	event, err := s.repo.GetEvent(ctx, eventId)
	if err != nil {
		return models.Events{}, fmt.Errorf("Service.UpdateEvent : %w", err)
	}

	if checkEventStatus(event.Status) != nil{
		return models.Events{}, fmt.Errorf("Service.UpdateEvent : %s", event.Status)
	}

	return s.repo.UpdateEvent(ctx, params, eventId)
}

func (s* EventService) LeaveEvent(ctx context.Context, callerId uuid.UUID, eventId uuid.UUID) (bool, error) {
	participant, err := s.repo.GetParticipant(ctx, callerId, eventId)
	if err != nil {
		return false, fmt.Errorf("Service.LeaveEvent : %w", err)
	}
	if participant.IsOwner {
		return false, fmt.Errorf("Service.LeaveEvent : owner can't leave")
	}

	_, err = s.repo.RemoveParticipant(ctx, callerId, eventId)
	if err != nil {
		return false, fmt.Errorf("Service.LeaveEvent : %w", err)
	}
	return true, nil
}