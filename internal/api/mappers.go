package api

import (
	"eventify-events/internal/models"
	v1 "eventify-events/pkg/api/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func eventToProto(event *models.Events) *v1.EventInfo {
	desc := ""
	if event.Description != nil{
		desc = *event.Description
	}
	
	eventResp := &v1.EventInfo{
		Id:           event.ID.String(),
		Title:        event.Title,
		Description:  desc,
		StartsAt:     timestamppb.New(event.StartsAt),
		Duration:     IntervalToMinutes(event.Duration),
		Status:       string(event.Status),
		IsPrivate:    event.IsPrivate,
	}
	return eventResp
}