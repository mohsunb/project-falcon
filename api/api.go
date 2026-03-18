package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
			if errors.Is(err, channels.ErrTooManyRequests) {
				w.Header().Set("Retry-After", "10")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("%v: try again later", err)})
				return
			}
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

	mux.HandleFunc("PATCH /channels/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		encoder := json.NewEncoder(w)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot parse id: %v", err)})
			return
		}
		var request channels.ChannelRepositionRequest
		json.NewDecoder(r.Body).Decode(&request)
		if err = channels.RepositionChannel(ctx, dbConnPool, id, request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			if errors.Is(err, channels.ErrChannelNotFound) {
				w.WriteHeader(http.StatusNotFound)
				encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot reposition channel: %v", err)})
				return
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				encoder.Encode(map[string]string{"message": fmt.Sprintf("failed to reposition channel: %v", err)})
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /channels/{channelID}/messages", func(w http.ResponseWriter, r *http.Request) {
		var request messages.MessageSendRequest
		json.NewDecoder(r.Body).Decode(&request)
		encoder := json.NewEncoder(w)
		channelID, err := uuid.Parse(r.PathValue("channelID"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot parse channel id: %v", err)})
			return
		}
		if err := messages.SaveMessage(ctx, dbConnPool, channelID, request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			if errors.Is(err, channels.ErrChannelNotFound) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
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
			encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot to parse channel id: %v", err)})
			return
		}

		cursorString := r.URL.Query().Get("cursor")
		var cursor uuid.UUID
		if len(cursorString) == 0 {
			cursor = uuid.Max
		} else {
			cursor, err = uuid.Parse(cursorString)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				encoder.Encode(map[string]string{"message": fmt.Sprintf("cannot to parse cursor: %v", err)})
				return
			}
		}

		pageSizeString := r.URL.Query().Get("page-size")
		if len(pageSizeString) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(map[string]string{"message": "page size must be specified"})
			return
		}
		pageSize, err := strconv.Atoi(pageSizeString)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(map[string]string{"message": "cannot parse page-size: must be an integer"})
			return
		}

		messages, err := messages.GetMessages(ctx, dbConnPool, channelID, cursor, pageSize)
		if err != nil {
			if errors.Is(err, channels.ErrChannelNotFound) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			encoder.Encode(map[string]string{"message": fmt.Sprint(err)})
			return
		}
		w.WriteHeader(http.StatusOK)
		encoder.Encode(messages)
	})
}
