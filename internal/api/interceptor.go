package api

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}
		userIDString := md.Get(UserIDMetadataKey)
		if len(userIDString) == 0 {
			return handler(ctx, req)
		}

		userID, err := uuid.Parse(userIDString[0])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid user id")
		}
		newCtx := ContextWithUserID(ctx, userID)

		return handler(newCtx, req)
	}
}