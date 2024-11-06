package config

import (
	"log/slog"
	"os"
	"time"
)

type Config struct {
	Server ServerConfig
	Logger LoggerConfig
}

type ServerConfig struct {
	Port            int
	Difficulty      int
	MaxConnections  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type LoggerConfig struct {
	Level     slog.Level
	AddSource bool
}

// DefaultConfig...
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8083,
			Difficulty:      24,
			MaxConnections:  1000,
			ReadTimeout:     time.Minute,
			WriteTimeout:    time.Minute,
			ShutdownTimeout: 5 * time.Second,
		},
		Logger: LoggerConfig{
			Level:     slog.LevelInfo,
			AddSource: true,
		},
	}
}

// SetupLogger...
func SetupLogger(cfg LoggerConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
