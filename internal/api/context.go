package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)
type contextKey string

const userIDKey contextKey = "user_id"

const UserIDMetadataKey = "x-user-id"


func ContextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("user_id not found in context")
	}
	return userID, nil
}