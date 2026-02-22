package messages

import (
	"context"
	"fmt"
	"project-falcon/channels"
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

func GetMessages(ctx context.Context, db *pgxpool.Pool, channelID uuid.UUID, cursor uuid.UUID, pageSize int) ([]Message, error) {
	channelExists, err := channels.ChannelExists(ctx, db, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if the channel exists: %w", err)
	}
	if !channelExists {
		return nil, fmt.Errorf("cannot fetch messages: %w", channels.ErrChannelNotFound)
	}

	rows, err := db.Query(ctx, "select id, message, creation_timestamp from messages where channel_id = $1 and id < $2 order by id desc limit $3", channelID, cursor, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get all messages: %w", err)
	}

	messages := make([]Message, 0)
	for rows.Next() {
		var message Message
		rows.Scan(&message.ID, &message.Message, &message.CreationTimestamp)
		messages = append(messages, message)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read all messages: %w", err)
	}

	return messages, nil
}

func SaveMessage(ctx context.Context, db *pgxpool.Pool, channelID uuid.UUID, request MessageSendRequest) error {
	channelExists, err := channels.ChannelExists(ctx, db, channelID)
	if err != nil {
		return fmt.Errorf("failed to check if the channel exists: %w", err)
	}
	if !channelExists {
		return fmt.Errorf("cannot save message: %w", channels.ErrChannelNotFound)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate a v7 uuid: %w", err)
	}

	if _, err := db.Exec(ctx, "insert into messages(id, message, creation_timestamp, channel_id) values ($1, $2, $3, $4)", id, request.Message, time.Now().UTC(), channelID); err != nil {
		return fmt.Errorf("failed to save the message: %w", err)
	}

	return nil
}
