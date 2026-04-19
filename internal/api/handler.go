package api

import (
	"context"
	"eventify-events/internal/models"
	"eventify-events/internal/services"
	"eventify-events/pkg/api/v1"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	microsecondsPerDay = 24 * 60 * 60 * 1000000
	microsecondsPerMinute = 60 * 1000000
	microsecondsPerMonth = 30 * microsecondsPerDay
)

type EventHandler struct {
    v1.UnimplementedEventServiceServer
    eventService     EventServiceInterface
    checklistService ChecklistServiceInterface
	log              *slog.Logger
}


func IntervalToMinutes(interval pgtype.Interval) int32 {
    // Стандартные значения
    totalMicroseconds := int64(interval.Months)*microsecondsPerMonth +
                         int64(interval.Days)*microsecondsPerDay +
                         interval.Microseconds
    // Сначала в time.Duration затем в минуты
    duration := time.Duration(totalMicroseconds) * time.Microsecond
    return int32(duration.Minutes())
}
func NewEventHandler(eventService EventServiceInterface, checklistService ChecklistServiceInterface, log *slog.Logger) *EventHandler {
	return &EventHandler{
		eventService:     eventService,
		checklistService: checklistService,
		log:              log,
	}
}

func (h *EventHandler) GetEvent(ctx context.Context, req *v1.GetEventRequest) (*v1.GetEventResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id")
	}
	event, err := h.eventService.GetEvent(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "event not found")
	}
	
	eventResp := &v1.GetEventResponse{
		Event: eventToProto(event),
	}
	return eventResp, nil
}

func (h *EventHandler) ListEvents(ctx context.Context, req *v1.ListEventsRequest) (*v1.ListEventsResponse, error) {
	var startsAfter *time.Time
	if req.StartsAfter != nil {
		t := req.StartsAfter.AsTime()
		startsAfter = &t
	}
	var startsBeforeFIlter *time.Time
	if req.StartsBefore != nil {
		t := req.StartsBefore.AsTime()
		startsBeforeFIlter = &t
	}

	filter := services.ListEventsFilter{
		Title: req.Title,
		Description: req.Description,
		StartsAfter: startsAfter,
		StartsBefore: startsBeforeFIlter,
		LocationName: req.LocationName,
	}

	events, err := h.eventService.ListEvents(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}
	response := make([]*v1.EventInfo, 0, len(events))
	for _, event := range events {
		response = append(response, eventToProto(&event))
	}
	return &v1.ListEventsResponse{Events: response}, nil
}

// Ожидает контекст с user_id, для этого использовать api/context.ContextWithUserID
func (h *EventHandler) ListUserEvents (ctx context.Context, req *v1.ListUserEventsRequest) (*v1.ListUserEventsResponse, error) {
	userId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	events, err := h.eventService.ListUserEvents(ctx, userId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user events: %v", err)
	}
	response := make([]*v1.EventInfo, 0, len(events))
	for _, event := range events {
		response = append(response, eventToProto(&event))
	}
	return &v1.ListUserEventsResponse{Events: response}, nil
}

func (h* EventHandler) CreateEvent(ctx context.Context, req *v1.CreateEventRequest) (*v1.CreateEventResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	duration := pgtype.Interval{
		Microseconds: int64(req.DurationMinutes * microsecondsPerMinute),
		Valid: true,
	}
	

	desc, locName := &req.Description, &req.LocationName
	if req.Description == "" {
		desc = nil
	}
	if req.LocationName == "" {
		locName = nil
	}

	locCoords, err := parseCoords(req.LocationCoords)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location coordinates")
	}
	maxParticipants := int(req.MaxParticipants)


	eventParams := &services.EventInputParams{
		IsPrivate: req.IsPrivate,
		Title: req.Title,
		Description: desc,
		StartsAt: req.StartsAt.AsTime(),
		Duration: duration,
		LocationName: locName,
		LocationCoords: locCoords,
		MaxParticipants: &maxParticipants,
	}
	event, err := h.eventService.CreateEvent(ctx, callerId, *eventParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}
	id := event.ID.String()
	response := &v1.CreateEventResponse{
		Id: id,
	}

	return response, nil
}

func (h* EventHandler) UpdateEvent(ctx context.Context, req *v1.UpdateEventRequest) (*v1.UpdateEventResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	var title, desc, locName *string
	if req.Title != nil {
		title = req.Title
	}
	if req.Description != nil {
		desc = req.Description
	}
	if req.LocationName != nil {
		locName = req.LocationName
	}
	locCoords, err := parseCoords(req.LocationCoords)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location coordinates")
	}
	var startsAt *time.Time
	if req.StartsAt != nil {
		t := req.StartsAt.AsTime()
		startsAt = &t
	}
	params := &models.UpdateEventParams{
		Title:          title,
		Description:    desc,
		StartsAt:       startsAt,
		LocationName:   locName,
		LocationCoords: locCoords,
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	
	event, err := h.eventService.UpdateEvent(ctx, callerId, eventId, *params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update event: %v", err)
	}
	eventInfo := eventToProto(&event)
	response := &v1.UpdateEventResponse{
		Event: eventInfo,
	}
	return response, nil
	
}

func (h* EventHandler) CancelEvent (ctx context.Context, req *v1.CancelEventRequest) (*v1.CancelEventResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	_, err = h.eventService.CancelEvent(ctx, callerId, eventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel event: %v", err)
	}
	return &v1.CancelEventResponse{Success: true}, nil
}

func (h* EventHandler) CreateInviteLink (ctx context.Context, req *v1.CreateInviteLinkRequest) (*v1.CreateInviteLinkResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		expiresAt = &t
	}
	
	code, err := h.eventService.CreateInviteLink(ctx, callerId, eventId, req.InviteType, expiresAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create invite link: %v", err)
	}
	return &v1.CreateInviteLinkResponse{EventCode: code}, nil
}

func (h* EventHandler) JoinEvent (ctx context.Context, req *v1.JoinEventRequest) (*v1.JoinEventResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}

	success, err := h.eventService.JoinEvent(ctx, callerId, req.EventCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to join event: %v", err)
	}
	return &v1.JoinEventResponse{Success: success}, nil
}

func (h* EventHandler) LeaveEvent (ctx context.Context, req *v1.LeaveEventRequest) (*v1.LeaveEventResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	success, err := h.eventService.LeaveEvent(ctx, callerId, eventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to leave event: %v", err)
	}
	return &v1.LeaveEventResponse{Success: success}, nil
}

func (h* EventHandler) GetEventParticipants (ctx context.Context, req *v1.GetEventParticipantsRequest) (*v1.GetEventParticipantsResponse, error) {
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	participants, err := h.eventService.GetEventParticipants(ctx, eventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get event participants: %v", err)
	}
	participantsInfo := participantsToProto(participants)

	response := &v1.GetEventParticipantsResponse{
		Participants: participantsInfo,
	}
	return response, nil
}

func (h* EventHandler) RemoveParticipant (ctx context.Context, req *v1.RemoveParticipantRequest) (*v1.RemoveParticipantResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	participantId, err := uuid.Parse(req.ParticipantId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid participant id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	success, err := h.eventService.RemoveParticipant(ctx, callerId, participantId, eventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove participant: %v", err)
	}
	return &v1.RemoveParticipantResponse{Success: success}, nil
}



// Checklist handlers

func (h* EventHandler) AddChecklistItem (ctx context.Context, req *v1.AddChecklistItemRequest) (*v1.AddChecklistItemResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	itemID, err := h.checklistService.AddChecklistItem(ctx, callerId, eventId, req.Title, int(req.Quantity), req.Unit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add checklist item: %v", err)
	}
	return &v1.AddChecklistItemResponse{ItemId: itemID.String()}, nil
}

func (h* EventHandler) RemoveChecklistItem (ctx context.Context, req *v1.RemoveChecklistItemRequest) (*v1.RemoveChecklistItemResponse, error){
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	itemId, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item id")
	}
	success, err := h.checklistService.RemoveChecklistItem(ctx, callerId, eventId, itemId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove checklist item: %v", err)
	}
	return &v1.RemoveChecklistItemResponse{Success: success}, nil
}

func (h* EventHandler) MarkItemPurchased (ctx context.Context, req *v1.MarkItemPurchasedRequest) (*v1.MarkItemPurchasedResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}
	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	itemId, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item id")
	}

	if req.BuyerId == nil {
		return nil, status.Errorf(codes.InvalidArgument, "buyer id is required")
	}

	var buyerId *uuid.UUID
	if req.BuyerId != nil {
		parsed, err := uuid.Parse(*req.BuyerId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid buyer id")
		}
		buyerId = &parsed
	}

	success, err := h.checklistService.MarkItemPurchased(ctx, callerId, eventId, itemId, buyerId, req.IsPurchased)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove checklist item: %v", err)
	}
	return &v1.MarkItemPurchasedResponse{Success: success}, nil
}

func (h* EventHandler) GetEventChecklist (ctx context.Context, req *v1.GetEventChecklistRequest) (*v1.GetEventChecklistResponse, error) {
	callerId, err := UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid user id")
	}

	eventId, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event id")
	}
	checklist, err := h.checklistService.GetEventChecklist(ctx, callerId, eventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get event checklist: %v", err)
	}
	checklistItems := checklistToProto(checklist)
	response := &v1.GetEventChecklistResponse{
		Checklist: checklistItems,
	}
	return response, nil
}