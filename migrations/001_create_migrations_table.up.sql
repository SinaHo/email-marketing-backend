-- Create a table to track which migrations have been applied
CREATE TABLE IF NOT EXISTS migrations (
    id          SERIAL PRIMARY KEY,
    version     VARCHAR(255) NOT NULL UNIQUE,
    run_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
