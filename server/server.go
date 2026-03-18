package server

import (
	"context"
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"project-falcon/api"
)

func PrepareServer(ctx context.Context, dbConnPool *pgxpool.Pool) *http.Server {
	mux := http.NewServeMux()
	api.RegisterEndpoints(mux, ctx, dbConnPool)
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return &server
}
