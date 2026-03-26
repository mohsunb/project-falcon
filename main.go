package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"project-falcon/database"
	"project-falcon/messages"
	"project-falcon/server"
)

func main() {
	signalCtx, signalCtxStop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer signalCtxStop()

	baseCtx, baseCtxStop := context.WithCancel(context.Background())
	pool, err := database.PrepareDatabase(baseCtx, "postgres", "password", "localhost", 5432, "postgres")
	if err != nil {
		log.Fatal(err)
	}

	server := server.PrepareServer(baseCtx, pool)
	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start the server: %v\n", err)
		}
	}()

	messagingHub := messages.MessagingHub
	go messagingHub.Run(baseCtx)

	<-signalCtx.Done()
	log.Println("shutdown initiated")
	err = server.Shutdown(baseCtx)
	baseCtxStop()
	if err != nil {
		log.Printf("could not shutdown the server: %v\n", err)
	}
	log.Println("closing db connection...")
	pool.Close()
	log.Println("successfully closed db connection")
	log.Println("shutdown complete")
}
