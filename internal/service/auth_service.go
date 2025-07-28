package service

import (
	"context"
	"errors"
	"time"

	"github.com/SinaHo/email-marketing-backend/api/v1/proto"
	"github.com/SinaHo/email-marketing-backend/internal/model"
	"github.com/SinaHo/email-marketing-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService defines business logic for authentication.
type AuthService interface {
	Register(ctx context.Context, in *proto.RegisterRequest) (*proto.RegisterResponse, error)
	Login(ctx context.Context, in *proto.LoginRequest) (*proto.LoginResponse, error)
}

type authService struct {
	repo        repository.UserRepository
	jwtSecret   []byte
	tokenExpiry time.Duration
}

// NewAuthService constructs a new AuthService.
func NewAuthService(repo repository.UserRepository, jwtSecret []byte, tokenExpiry time.Duration) AuthService {
	return &authService{
		repo:        repo,
		jwtSecret:   jwtSecret,
		tokenExpiry: tokenExpiry,
	}
}

// Register implements the Register RPC: it creates a new user, hashes password, saves to DB, and returns JWT.
func (s *authService) Register(ctx context.Context, in *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	// 1. Basic validation
	if in.Email == "" || in.Password == "" {
		return nil, errors.New("email and password are required")
	}

	// 2. Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// 3. Convert proto.Language to model.Language
	var lang model.Language
	switch in.Lang {
	case proto.RegisterRequest_EN:
		lang = model.Language_EN
	case proto.RegisterRequest_FA:
		lang = model.Language_FA
	default:
		lang = model.Language_EN
	}

	// 4. Insert into repository
	u, err := s.repo.Create(ctx, in.Email, string(hashed), lang, in.ReferrerCode)
	if err != nil {
		return nil, err
	}

	// 5. Generate JWT
	//    - We put user ID and email in the token claims; adjust as you like.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   u.ID.String(),
		"email": u.Email,
		"exp":   time.Now().Add(s.tokenExpiry).Unix(),
	})
	jwtStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to sign JWT")
	}

	return &proto.RegisterResponse{
		Id:           u.ID.String(),
		ReferralCode: u.ReferralCode,
		JwtToken:     jwtStr,
	}, nil
}

// Login implements the Login RPC: it verifies email+password, then returns a fresh JWT.
func (s *authService) Login(ctx context.Context, in *proto.LoginRequest) (*proto.LoginResponse, error) {
	if in.Email == "" || in.Password == "" {
		return nil, errors.New("email and password are required")
	}

	// 1. Fetch user by email
	u, err := s.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("invalid email or password")
	}

	// 2. Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// 3. Generate new JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   u.ID.String(),
		"email": u.Email,
		"exp":   time.Now().Add(s.tokenExpiry).Unix(),
	})
	jwtStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to sign JWT")
	}

	return &proto.LoginResponse{
		JwtCode: jwtStr,
	}, nil
}
