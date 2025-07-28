package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SinaHo/email-marketing-backend/api/v1/proto"
	"github.com/SinaHo/email-marketing-backend/internal/model"
	"github.com/SinaHo/email-marketing-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo implements repository.UserRepository for unit testing
type mockUserRepo struct {
	// capture inputs
	createdEmail        string
	createdPasswordHash string
	createdLang         model.Language
	createdReferrer     int32
	// control outputs
	createResult    *model.User
	createError     error
	getByEmailInput string
	getByEmailUser  *model.User
	getByEmailError error
}

func (m *mockUserRepo) Create(ctx context.Context, email, passwordHash string, lang model.Language, referrerCode int32) (*model.User, error) {
	m.createdEmail = email
	m.createdPasswordHash = passwordHash
	m.createdLang = lang
	m.createdReferrer = referrerCode
	if m.createError != nil {
		return nil, m.createError
	}
	return m.createResult, nil
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	m.getByEmailInput = email
	if m.getByEmailError != nil {
		return nil, m.getByEmailError
	}
	return m.getByEmailUser, nil
}
func TestRegister_Success(t *testing.T) {
	ctx := context.Background()
	// logger := zap.NewNop()
	// Prepare a mock repository that returns a simple User
	newID := "123e4567-e89b-12d3-a456-426655440000"
	mockUser := &model.User{
		ID:           uuid.MustParse(newID),
		Email:        "alice@example.com",
		PasswordHash: "", // irrelevant
		Lang:         model.Language_EN,
		ReferralCode: "abcd1234",
		ReferrerCode: 42,
		CreatedAt:    time.Now(),
	}
	repo := &mockUserRepo{
		createResult: mockUser,
		createError:  nil,
	}
	authSvc := service.NewAuthService(repo, []byte("test-secret"), time.Hour)
	// 1) Successful register
	req := &proto.RegisterRequest{
		Email:        "alice@example.com",
		Password:     "password123",
		Lang:         proto.RegisterRequest_EN,
		ReferrerCode: 42,
	}
	resp, err := authSvc.Register(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, newID, resp.Id)
	assert.Equal(t, "abcd1234", resp.ReferralCode)
	assert.NotEmpty(t, resp.JwtToken)
	// Check that the repo.Create was called with expected values:
	assert.Equal(t, "alice@example.com", repo.createdEmail)
	// The password hash should match when compared via bcrypt:
	err = bcrypt.CompareHashAndPassword([]byte(repo.createdPasswordHash), []byte("password123"))
	assert.NoError(t, err)
	assert.Equal(t, model.Language_EN, repo.createdLang)
	assert.Equal(t, int32(42), repo.createdReferrer)
}
func TestRegister_MissingEmailOrPassword(t *testing.T) {
	ctx := context.Background()
	repo := &mockUserRepo{}
	authSvc := service.NewAuthService(repo, []byte("secret"), time.Hour)
	// Missing email
	_, err := authSvc.Register(ctx, &proto.RegisterRequest{
		Email:        "",
		Password:     "",
		Lang:         proto.RegisterRequest_EN,
		ReferrerCode: 0,
	})
	assert.Error(t, err)
	// Missing password
	_, err = authSvc.Register(ctx, &proto.RegisterRequest{
		Email:        "bob@example.com",
		Password:     "",
		Lang:         proto.RegisterRequest_EN,
		ReferrerCode: 0,
	})
	assert.Error(t, err)
}
func TestRegister_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := &mockUserRepo{
		createError: errors.New("something went wrong"),
	}
	authSvc := service.NewAuthService(repo, []byte("secret"), time.Hour)
	_, err := authSvc.Register(ctx, &proto.RegisterRequest{
		Email:        "charlie@example.com",
		Password:     "pass",
		Lang:         proto.RegisterRequest_FA,
		ReferrerCode: 7,
	})
	assert.Error(t, err)
}
func TestLogin_Success(t *testing.T) {
	ctx := context.Background()
	// logger := zap.NewNop()
	// Hash a known password
	hashed, _ := bcrypt.GenerateFromPassword([]byte("mysecurepass"), bcrypt.DefaultCost)
	existingUser := &model.User{
		ID:           uuid.MustParse("223e4567-e89b-12d3-a456-426655440000"),
		Email:        "dana@example.com",
		PasswordHash: string(hashed),
		Lang:         model.Language_EN,
		ReferralCode: "zzz999",
		ReferrerCode: 11,
		CreatedAt:    time.Now(),
	}
	repo := &mockUserRepo{
		getByEmailUser:  existingUser,
		getByEmailError: nil,
	}
	authSvc := service.NewAuthService(repo, []byte("another-secret"), time.Hour)
	req := &proto.LoginRequest{
		Email:    "dana@example.com",
		Password: "mysecurepass",
	}
	resp, err := authSvc.Login(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.JwtCode)
	// If we tamper password, should error
	_, err = authSvc.Login(ctx, &proto.LoginRequest{
		Email:    "dana@example.com",
		Password: "wrongpass",
	})
	assert.Error(t, err)
}
func TestLogin_UserNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockUserRepo{
		getByEmailUser:  nil,
		getByEmailError: nil,
	}
	authSvc := service.NewAuthService(repo, []byte("secret"), time.Hour)
	_, err := authSvc.Login(ctx, &proto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "whatever",
	})
	assert.Error(t, err)
}
func TestLogin_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := &mockUserRepo{
		getByEmailError: errors.New("db issue"),
	}
	authSvc := service.NewAuthService(repo, []byte("secret"), time.Hour)
	_, err := authSvc.Login(ctx, &proto.LoginRequest{
		Email:    "error@example.com",
		Password: "pass",
	})
	assert.Error(t, err)
}
