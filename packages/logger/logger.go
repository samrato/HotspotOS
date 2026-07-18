package logger

import (
	"log/slog"
	"os"
	"sync"
)

var (
	log  *slog.Logger
	once sync.Once
)

// Init initializes the global logger
func Init(env string) {
	once.Do(func() {
		var handler slog.Handler
		if env == "production" {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
		} else {
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})
		}
		log = slog.New(handler)
		slog.SetDefault(log)
	})
}

// Info logs a message at Info level
func Info(msg string, args ...any) {
	if log == nil {
		Init("development")
	}
	log.Info(msg, args...)
}

// Debug logs a message at Debug level
func Debug(msg string, args ...any) {
	if log == nil {
		Init("development")
	}
	log.Debug(msg, args...)
}

// Warn logs a message at Warn level
func Warn(msg string, args ...any) {
	if log == nil {
		Init("development")
	}
	log.Warn(msg, args...)
}

// Error logs a message at Error level
func Error(msg string, args ...any) {
	if log == nil {
		Init("development")
	}
	log.Error(msg, args...)
}
