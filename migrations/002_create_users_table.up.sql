CREATE TABLE IF NOT EXISTS users (
    id             UUID PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    lang           INTEGER NOT NULL,
    referral_code  TEXT NOT NULL UNIQUE,
    referrer_code  INTEGER NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
