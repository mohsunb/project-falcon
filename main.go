package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"project-falcon/config"
	"project-falcon/database"
	"project-falcon/messages"
	"project-falcon/server"
)

func main() {
	config := config.ParseConfig()
	logger := prepareLogger(config)
	signalCtx, signalCtxStop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCtxStop()

	baseCtx, baseCtxStop := context.WithCancel(context.Background())
	pool, err := database.PrepareDatabase(baseCtx, logger, config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Name)
	if err != nil {
		logger.Error("failed to prepare the database", "error", err)
		os.Exit(1)
	} else {
		logger.Info("database is ready")
	}

	server := server.PrepareServer(baseCtx, pool, logger)
	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start the server", "error", err)
			os.Exit(1)
		}
	}()

	messagingHub := messages.MessagingHub
	go messagingHub.Run(baseCtx)

	<-signalCtx.Done()
	logger.Info("shutdown initiated")
	err = server.Shutdown(baseCtx)
	baseCtxStop()
	if err != nil {
		logger.Error("could not shutdown the server", "error", err)
	}
	logger.Info("closing db connection...")
	pool.Close()
	logger.Info("successfully closed db connection")
	logger.Info("shutdown complete")
}

func prepareLogger(config config.Config) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(strings.TrimSpace(config.Log.Level)) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	handlerOptions := &slog.HandlerOptions{Level: logLevel}
	var logHandler slog.Handler
	if strings.ToLower(strings.TrimSpace(config.Log.Type)) == "json" {
		logHandler = slog.NewJSONHandler(os.Stderr, handlerOptions)
	} else {
		logHandler = slog.NewTextHandler(os.Stderr, handlerOptions)
	}
	return slog.New(logHandler)
}
