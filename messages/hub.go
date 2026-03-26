package messages

import (
	"context"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

var MessagingHub *Hub = newHub()

type Hub struct {
	channels   map[uuid.UUID]map[*websocket.Conn]struct{}
	register   chan registration
	unregister chan registration
	broadcast  chan broadcast
}

type registration struct {
	channelID uuid.UUID
	client    *websocket.Conn
}

type broadcast struct {
	channelID uuid.UUID
	message   []byte
}

func newHub() *Hub {
	return &Hub{
		channels:   map[uuid.UUID]map[*websocket.Conn]struct{}{},
		register:   make(chan registration),
		unregister: make(chan registration),
		broadcast:  make(chan broadcast, 256),
	}
}

func (hub *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for channelID := range hub.channels {
				for client := range hub.channels[channelID] {
					client.CloseNow()
				}
			}
			return
		case r := <-hub.register:
			if _, channelRegistered := hub.channels[r.channelID]; !channelRegistered {
				hub.channels[r.channelID] = make(map[*websocket.Conn]struct{})
			}
			hub.channels[r.channelID][r.client] = struct{}{}
		case r := <-hub.unregister:
			delete(hub.channels[r.channelID], r.client)
			if len(hub.channels[r.channelID]) == 0 {
				delete(hub.channels, r.channelID)
			}
		case b := <-hub.broadcast:
			if _, channelRegistered := hub.channels[b.channelID]; channelRegistered {
				for client := range hub.channels[b.channelID] {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					if err := client.Write(ctx, websocket.MessageText, b.message); err != nil {
						delete(hub.channels[b.channelID], client)
						// TODO: make sure this has some value
						client.CloseNow()
					}
					cancel()
				}
				if len(hub.channels[b.channelID]) == 0 {
					delete(hub.channels, b.channelID)
				}
			}
		}
	}
}

func (hub *Hub) Register(channelID uuid.UUID, client *websocket.Conn) {
	hub.register <- registration{
		channelID: channelID,
		client:    client,
	}
}

func (hub *Hub) Unregister(channelID uuid.UUID, client *websocket.Conn) {
	hub.unregister <- registration{
		channelID: channelID,
		client:    client,
	}
}

func (hub *Hub) Broadcast(channelID uuid.UUID, message []byte) {
	hub.broadcast <- broadcast{
		channelID: channelID,
		message:   message,
	}
}
