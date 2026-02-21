package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID                uuid.UUID `json:"id"`
	Message           string    `json:"message"`
	CreationTimestamp time.Time `json:"timestamp"`
}

type MessageSendRequest struct {
	Message string `json:"message"`
}

func GetAllMessages(ctx context.Context, pool *pgxpool.Pool, channelID uuid.UUID) ([]Message, error) {
	rows, err := pool.Query(ctx, "select id, message, creation_timestamp from messages where channel_id = $1", channelID)
	if err != nil {
		return nil, fmt.Errorf(`failed to get all messages: %v`, err)
	}

	messages := make([]Message, 0)
	for rows.Next() {
		var message Message
		rows.Scan(&message.ID, &message.Message, &message.CreationTimestamp)
		messages = append(messages, message)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read all messages: %v", err)
	}

	return messages, nil
}

func SaveMessage(ctx context.Context, pool *pgxpool.Pool, channelID uuid.UUID, request MessageSendRequest) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate a v7 uuid: %v", err)
	}

	if _, err := pool.Exec(ctx, "insert into messages(id, message, creation_timestamp, channel_id) values ($1, $2, $3, $4)", id, request.Message, time.Now().UTC(), channelID); err != nil {
		return fmt.Errorf("failed to save the message: %v", err)
	}

	return nil
}
