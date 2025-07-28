package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/SinaHo/email-marketing-backend/api/v1/proto"
	"github.com/SinaHo/email-marketing-backend/internal/handler"
)

// mockAuthService implements the AuthService interface for handler tests.
type mockAuthService struct {
	// Control fields to decide what to return
	registerResponse *proto.RegisterResponse
	registerError    error

	loginResponse *proto.LoginResponse
	loginError    error
}

func (m *mockAuthService) Register(ctx context.Context, in *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	if m.registerError != nil {
		return nil, m.registerError
	}
	return m.registerResponse, nil
}

func (m *mockAuthService) Login(ctx context.Context, in *proto.LoginRequest) (*proto.LoginResponse, error) {
	if m.loginError != nil {
		return nil, m.loginError
	}
	return m.loginResponse, nil
}

func TestAuthHandler_Register_Success(t *testing.T) {
	// logger := zap.NewNop()
	expected := &proto.RegisterResponse{
		Id:           "some-id",
		ReferralCode: "ABCD1234",
		JwtToken:     "jwt-token-string",
	}
	mockSvc := &mockAuthService{
		registerResponse: expected,
		registerError:    nil,
	}
	h := handler.NewAuthHandler(mockSvc)

	resp, err := h.Register(context.Background(), &proto.RegisterRequest{
		Email:        "test@example.com",
		Lang:         proto.RegisterRequest_EN,
		Password:     "pass",
		ReferrerCode: 5,
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, resp)
}

func TestAuthHandler_Register_Error(t *testing.T) {
	// logger := zap.NewNop()
	mockSvc := &mockAuthService{
		registerResponse: nil,
		registerError:    errors.New("service failure"),
	}
	h := handler.NewAuthHandler(mockSvc)

	resp, err := h.Register(context.Background(), &proto.RegisterRequest{
		Email:        "fail@example.com",
		Lang:         proto.RegisterRequest_EN,
		Password:     "pass",
		ReferrerCode: 5,
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	// logger := zap.NewNop()
	expected := &proto.LoginResponse{JwtCode: "new-jwt"}
	mockSvc := &mockAuthService{
		loginResponse: expected,
		loginError:    nil,
	}
	h := handler.NewAuthHandler(mockSvc)

	resp, err := h.Login(context.Background(), &proto.LoginRequest{
		Email:    "someone@example.com",
		Password: "pwd",
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, resp)
}

func TestAuthHandler_Login_Error(t *testing.T) {
	// logger := zap.NewNop()
	mockSvc := &mockAuthService{
		loginResponse: nil,
		loginError:    errors.New("service error"),
	}
	h := handler.NewAuthHandler(mockSvc)

	resp, err := h.Login(context.Background(), &proto.LoginRequest{
		Email:    "someone@example.com",
		Password: "pwd",
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
}
