package api

import (
	"eventify-events/internal/models"
	v1 "eventify-events/pkg/api/v1"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
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

func participantsToProto(participants []models.EventParticipants) ([]*v1.ParticipantInfo) {
	participantsInfo := make([]*v1.ParticipantInfo, 0, len(participants))
	for _, p := range participants{
		role, status := "", ""
		if p.Role != nil {
			role = *p.Role
		}
		if p.Status != "" {
			status = string(p.Status)
		}
		participantsInfo = append(participantsInfo, &v1.ParticipantInfo{
			ParticipantId: p.UserID.String(),
			Role:          role,
			Status:        status,
		})
	}
	return participantsInfo
}

func checklistToProto(items []models.ChecklistItems) []*v1.ChecklistItemInfo {
    result := make([]*v1.ChecklistItemInfo, 0, len(items))
    for _, item := range items {
        unit := ""
        if item.Unit != nil {
            unit = *item.Unit
        }
        result = append(result, &v1.ChecklistItemInfo{
            Id:          item.ID.String(),
            Title:       item.Title,
            Quantity:    int32(item.Quantity),
            Unit:        unit,
            IsPurchased: &item.IsPurchased,
        })
    }
    return result
}
func parseCoords(coords *string) (*pgtype.Point, error) {
	if coords == nil {
		return nil, nil
	}

	latLongCoords := strings.Split(*coords, ",")
	if len(latLongCoords) != 2 {
		return nil, nil
	}
	lat, err := strconv.ParseFloat(latLongCoords[0], 64)
	if err != nil {
		return nil, err
	}
	long, err := strconv.ParseFloat(latLongCoords[1], 64)
	if err != nil {
		return nil, err
	}
	point := &pgtype.Point{
		P: pgtype.Vec2{
			X: lat,
			Y: long,
		},
		Valid: true,
	}
	return point, nil
}