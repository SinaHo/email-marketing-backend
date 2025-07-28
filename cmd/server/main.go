// cmd/server/main.go
package main

import (
	"os"
	"os/signal"
	"syscall"

	"net/http"
	_ "net/http/pprof"

	"github.com/SinaHo/email-marketing-backend/internal/config"
	"github.com/SinaHo/email-marketing-backend/internal/server"
	"go.uber.org/zap"
)

func main() {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize zap logger: " + err.Error())
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig("internal/config")
	if err != nil {
		logger.Sugar().Fatalf("failed to load config: %v", err)
	}

	// Create AppServer with zap logger
	app, err := server.NewAppServer(cfg, logger)
	if err != nil {
		logger.Sugar().Fatalf("failed to initialize server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := app.Run(); err != nil {
			logger.Sugar().Fatalf("server run error: %v", err)
		}
	}()
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	// Wait for interrupt (SIGINT/SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Sugar().Info("Received shutdown signal")
	app.GracefulStop()
	logger.Sugar().Info("Server stopped")
}
