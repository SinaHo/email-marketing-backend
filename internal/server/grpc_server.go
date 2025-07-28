// internal/server/grpc_server.go
package server

import (
	"fmt"
	"net"
	"time"

	"github.com/SinaHo/email-marketing-backend/api/v1/proto"
	"github.com/SinaHo/email-marketing-backend/internal/config"
	"github.com/SinaHo/email-marketing-backend/internal/handler"
	"github.com/SinaHo/email-marketing-backend/internal/middleware"
	"github.com/SinaHo/email-marketing-backend/internal/repository"
	"github.com/SinaHo/email-marketing-backend/internal/service"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/lib/pq"
)

type AppServer struct {
	cfg    *config.Config
	logger *zap.Logger
	db     *sqlx.DB
	rdb    *redis.Client
	GRPC   *grpc.Server
}

func NewAppServer(cfg *config.Config, logger *zap.Logger) (*AppServer, error) {
	sugar := logger.Sugar()

	// PostgreSQL (via sqlx)
	pg := cfg.Postgres
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pg.Host, pg.Port, pg.User, pg.Password, pg.DBName, pg.SSLMode,
	)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		sugar.Errorf("failed to connect to postgres: %v", err)
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	// Redis
	// rd := cfg.Redis
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     rd.Addr,
	// 	Password: rd.Password,
	// 	DB:       rd.DB,
	// })
	// if _, err := rdb.Ping(rdb.Context()).Result(); err != nil {
	// 	sugar.Errorf("failed to ping redis: %v", err)
	// 	return nil, fmt.Errorf("redis ping: %w", err)
	// }

	// Logging interceptor & Auth interceptor
	authInt := middleware.AuthInterceptor(sugar, cfg.JWT.SigningKey)
	logInt := middleware.UnaryLoggingInterceptor(sugar)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logInt, authInt),
	)

	// Repository → Service → Handler
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewAuthService(userRepo, []byte(cfg.JWT.SigningKey), 1*time.Hour)
	userHandler := handler.NewAuthHandler(userSvc)

	proto.RegisterAuthenticationServer(grpcServer, userHandler)
	reflection.Register(grpcServer)

	sugar.Infof("AppServer initialized successfully")
	return &AppServer{
		cfg:    cfg,
		logger: logger,
		db:     db,
		// rdb:    rdb,
		GRPC: grpcServer,
	}, nil
}

func (a *AppServer) Run() error {
	sugar := a.logger.Sugar()
	addr := fmt.Sprintf(":%d", a.cfg.Server.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		sugar.Errorf("listen error on %s: %v", addr, err)
		return fmt.Errorf("listen: %w", err)
	}
	sugar.Infof("gRPC server listening on %s", addr)
	return a.GRPC.Serve(lis)
}

func (a *AppServer) GracefulStop() {
	sugar := a.logger.Sugar()
	sugar.Info("Shutting down gRPC server gracefully")
	a.GRPC.GracefulStop()
	a.db.Close()
	a.rdb.Close()
	sugar.Info("Resources closed, server stopped")
}
