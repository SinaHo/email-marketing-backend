package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/SinaHo/email-marketing-backend/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// UserRepository defines the methods we need for storing and retrieving users.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string, lang model.Language, referrerCode int32) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository constructs a new UserRepository backed by a sqlx.DB.
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new User into PostgreSQL.
// It generates a new UUID, a referral code, and sets CreatedAt to now.
func (r *userRepository) Create(
	ctx context.Context,
	email, passwordHash string,
	lang model.Language,
	referrerCode int32,
) (*model.User, error) {
	// 1. Ensure no existing user with same email:
	var exists bool
	err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email)
	if err != nil {
		return nil, fmt.Errorf("error checking existing email: %w", err)
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	// 2. Generate new UUID
	id := uuid.New()
	// 3. Generate a referral code string (for simplicity, just use first 8 chars of UUID)
	refCode := id.String()[:8]

	// 4. Set CreatedAt
	createdAt := time.Now().UTC()

	// 5. Insert into DB
	query := `
		INSERT INTO users (
			id, email, password_hash, lang, referral_code, referrer_code, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password_hash, lang, referral_code, referrer_code, created_at
	`
	var u model.User
	err = r.db.GetContext(
		ctx,
		&u,
		query,
		id,
		email,
		passwordHash,
		int32(lang),
		refCode,
		referrerCode,
		createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error inserting user: %w", err)
	}
	return &u, nil
}

// GetByEmail fetches a user row by its email. Returns (nil, nil) if not found.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	query := `
		SELECT id, email, password_hash, lang, referral_code, referrer_code, created_at
		FROM users
		WHERE email = $1
	`
	err := r.db.GetContext(ctx, &u, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error selecting user by email: %w", err)
	}
	return &u, nil
}
