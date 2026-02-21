package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"project-falcon/channels"
	"project-falcon/messages"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterEndpoints(mux *http.ServeMux, ctx context.Context, dbConnPool *pgxpool.Pool) {
	mux.HandleFunc("POST /channels", func(w http.ResponseWriter, r *http.Request) {
		var request channels.ChannelCreationRequest
		json.NewDecoder(r.Body).Decode(&request)
		if err := channels.CreateChannel(ctx, dbConnPool, request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprint(err)})
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /channels", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		channels, err := channels.GetAllChannels(ctx, dbConnPool)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode(map[string]string{"message": fmt.Sprint(err)})
			return
		}
		w.WriteHeader(http.StatusOK)
		encoder.Encode(channels)
	})
	mux.HandleFunc("POST /channels/{channelID}/messages", func(w http.ResponseWriter, r *http.Request) {
		var request messages.MessageSendRequest
		json.NewDecoder(r.Body).Decode(&request)
		encoder := json.NewEncoder(w)
		channelID, err := uuid.Parse(r.PathValue("channelID"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot to parse channelID: %v", err)})
		}
		if err := messages.SaveMessage(ctx, dbConnPool, channelID, request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode(map[string]string{"message": fmt.Sprint(err)})
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /channels/{channelID}/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		channelID, err := uuid.Parse(r.PathValue("channelID"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot to parse channelID: %v", err)})
		}
		messages, err := messages.GetAllMessages(ctx, dbConnPool, channelID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode(map[string]string{"message": fmt.Sprint(err)})
			return
		}
		w.WriteHeader(http.StatusOK)
		encoder.Encode(messages)
	})
}
