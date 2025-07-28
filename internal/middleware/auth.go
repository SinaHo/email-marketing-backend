package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ErrUnauthenticated is returned when no or invalid token is provided.
var ErrUnauthenticated = status.Errorf(codes.Unauthenticated, "unauthenticated")

// AuthInterceptor returns a unary interceptor that checks for a valid JWT.
func AuthInterceptor(logger *zap.SugaredLogger, jwtSecret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip health check or other public methods if needed:
		// if info.FullMethod == "/grpc.health.v1.Health/Check" {
		//     return handler(ctx, req)
		// }

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("Missing metadata in context")
			return nil, ErrUnauthenticated
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			logger.Warn("No authorization header provided")
			return nil, ErrUnauthenticated
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == "" {
			logger.Warn("Empty bearer token")
			return nil, ErrUnauthenticated
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Check signing method etc.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			logger.Warnw("Invalid token", "error", err)
			return nil, ErrUnauthenticated
		}

		// Optionally, you can extract claims and add to context.
		// if claims, ok := token.Claims.(jwt.MapClaims); ok {
		//     ctx = context.WithValue(ctx, "user_id", claims["sub"])
		// }

		return handler(ctx, req)
	}
}
