package integration_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/SinaHo/email-marketing-backend/api/v1/proto"
	"github.com/SinaHo/email-marketing-backend/internal/config"
	"github.com/SinaHo/email-marketing-backend/internal/server"
)

// TestIntegration_RegisterAndLogin runs the full gRPC server and exercises Register & Login.
func TestIntegration_RegisterAndLogin(t *testing.T) {
	// 1. Use zap.NewNop() for test logging
	logger := zap.NewNop()

	// 2. Load config (will pick up ENV overrides, e.g. POSTGRES_*, REDIS_*, JWT_SIGNING_KEY, etc.)
	cfg, err := config.LoadConfig("../configs")
	assert.NoError(t, err)

	// 3. (Re-)create the “users” table afresh for a clean test state
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User,
		cfg.Postgres.Password, cfg.Postgres.DBName, cfg.Postgres.SSLMode,
	)
	db, err := sqlx.Connect("postgres", dsn)
	assert.NoError(t, err)
	defer db.Close()

	// Clean up users table
	_, err = db.Exec(`DROP TABLE IF EXISTS users;`)
	assert.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			lang INTEGER NOT NULL,
			referral_code TEXT NOT NULL UNIQUE,
			referrer_code INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	assert.NoError(t, err)

	// Also ensure migrations table exists so service can record applied migrations if used
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) NOT NULL UNIQUE,
			run_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	assert.NoError(t, err)

	// 4. Start gRPC server on a random free port
	lis, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	addr := lis.Addr().String()

	app, err := server.NewAppServer(cfg, logger)
	assert.NoError(t, err)

	grpcServer := app.GRPC // assume we exposed the grpc.Server for testing
	go func() {
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			panic(fmt.Sprintf("gRPC serve error: %v", serveErr))
		}
	}()
	defer func() {
		app.GracefulStop()
	}()

	// 5. Dial the server
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	client := proto.NewAuthenticationClient(conn)

	// 6. Call Register
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	registerReq := &proto.RegisterRequest{
		Email:        "inttest@example.com",
		Lang:         proto.RegisterRequest_EN,
		Password:     "integrationPass123",
		ReferrerCode: 99,
	}
	regResp, err := client.Register(ctx, registerReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, regResp.Id)
	assert.NotEmpty(t, regResp.ReferralCode)
	assert.NotEmpty(t, regResp.JwtToken)

	// 7. Confirm the user exists in the DB
	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM users WHERE email = $1;`, "inttest@example.com")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 8. Call Login (should return a fresh JWT)
	loginReq := &proto.LoginRequest{
		Email:    "inttest@example.com",
		Password: "integrationPass123",
	}
	loginResp, err := client.Login(ctx, loginReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResp.JwtCode)

	// 9. Attempt login with wrong password (should error)
	_, err = client.Login(ctx, &proto.LoginRequest{
		Email:    "inttest@example.com",
		Password: "wrongPassword",
	})
	assert.Error(t, err)
}
