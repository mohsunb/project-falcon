package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"project-falcon/database"
	"project-falcon/messages"
	"project-falcon/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	signalCtx, signalCtxStop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCtxStop()

	baseCtx, baseCtxStop := context.WithCancel(context.Background())
	pool, err := database.PrepareDatabase(baseCtx, logger, "postgres", "password", "localhost", 5432, "postgres")
	if err != nil {
		logger.Error("failed to prepare the database", "error", err)
		os.Exit(1)
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
