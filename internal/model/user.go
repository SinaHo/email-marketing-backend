package model

import (
	"time"

	"github.com/google/uuid"
)

type Language int32

const (
	Language_EN Language = 0
	Language_FA Language = 1
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Lang         Language  `db:"lang"`
	ReferralCode string    `db:"referral_code"`
	ReferrerCode int32     `db:"referrer_code"`
	CreatedAt    time.Time `db:"created_at"`
}
