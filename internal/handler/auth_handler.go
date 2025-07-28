package handler

import (
	"context"

	"github.com/SinaHo/email-marketing-backend/api/v1/proto"
	"github.com/SinaHo/email-marketing-backend/internal/service"
)

// AuthHandler is the gRPC server implementation of Authentication service.
type AuthHandler struct {
	proto.UnimplementedAuthenticationServer
	svc service.AuthService
}

// NewAuthHandler constructs a new handler, given an AuthService.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	return h.svc.Register(ctx, req)
}

func (h *AuthHandler) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	return h.svc.Login(ctx, req)
}
