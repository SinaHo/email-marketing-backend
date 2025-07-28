// internal/logger/logger.go
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(levelStr, format string) (*zap.Logger, error) {
	var cfg zap.Config
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	// Parse level
	lvl := zapcore.InfoLevel
	err := lvl.UnmarshalText([]byte(levelStr))
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	return cfg.Build(zap.AddCaller())
}

// Sugar is a convenience wrapper
func NewSugar(levelStr, format string) (*zap.SugaredLogger, error) {
	base, err := NewLogger(levelStr, format)
	if err != nil {
		return nil, err
	}
	return base.Sugar(), nil
}
