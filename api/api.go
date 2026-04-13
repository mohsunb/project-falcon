package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"project-falcon/channels"
	"project-falcon/messages"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterEndpoints(mux *http.ServeMux, ctx context.Context, dbConnPool *pgxpool.Pool, logger *slog.Logger) {
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
		if err = channels.RepositionChannel(ctx, dbConnPool, logger, id, request); err != nil {
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

	mux.HandleFunc("GET /channels/{channelID}/messages/live", func(w http.ResponseWriter, r *http.Request) {
		channelID, err := uuid.Parse(r.PathValue("channelID"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("cannot open socket: cannot parse channel id: %v", err)})
			return
		}

		channelExists, err := channels.ChannelExists(ctx, dbConnPool, channelID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("cannot open socket: failed to determine if the channel exists: %v", err)})
			return
		}

		if !channelExists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("cannot open socket: %v", channels.ErrChannelNotFound)})
			return
		}

		socket, err := websocket.Accept(w, r, nil)
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to upgrade the connection to websocket", "error", err)
			return
		}
		messages.MessagingHub.Register(channelID, socket)
		key := r.Header.Get("Sec-WebSocket-Key")
		logger.DebugContext(r.Context(), "established websocket connection", "key", key)
		defer func() {
			messages.MessagingHub.Unregister(channelID, socket)
			socket.CloseNow()
			logger.DebugContext(r.Context(), "closed websocket connection", "key", key)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, msg, err := socket.Read(ctx)
			if err != nil {
				return
			}
			messages.SaveMessageUnchecked(ctx, dbConnPool, channelID, messages.MessageSendRequest{Message: string(msg)})
		}
	})
}
