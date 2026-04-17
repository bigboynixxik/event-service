package api

import (
	"context"
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
	events, err := h.eventService.ListEvents(ctx, services.ListEventsFilter{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}
	response := make([]*v1.EventInfo, 0, len(events))
	for _, event := range events {
		response = append(response, eventToProto(&event))
	}
	return &v1.ListEventsResponse{Events: response}, nil
}