package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"project-falcon/api"

	"github.com/jackc/pgx/v5/pgxpool"
)

func PrepareServer(ctx context.Context, dbConnPool *pgxpool.Pool, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	api.RegisterEndpoints(mux, ctx, dbConnPool, logger)
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return &server
}
