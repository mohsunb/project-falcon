package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Channel struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
}

type ChannelCreationRequest struct {
	Name string `json:"name"`
}

func CreateChannel(ctx context.Context, conn *pgxpool.Pool, request ChannelCreationRequest) error {
	query := "insert into channels (id, name, creation_timestamp) values ($1, $2, $3)"
	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate a random id for the channel: %w", err)
	}
	if _, err := conn.Exec(ctx, query,
		id,
		request.Name,
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("failed to create the channel: %w", err)
	}
	return nil
}

func GetAllChannels(ctx context.Context, conn *pgxpool.Pool) ([]Channel, error) {
	channels := make([]Channel, 0)
	rows, err := conn.Query(ctx, "select id, name, creation_timestamp from channels")
	if err != nil {
		return nil, fmt.Errorf("failed to get all channels: %w", err)
	}

	for rows.Next() {
		var channel Channel
		rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.CreationTimestamp,
		)
		channels = append(channels, channel)
	}

	if rows.Err() != nil {
		return channels, fmt.Errorf("failed to read all channels: %w", err)
	}

	return channels, nil
}

func ChannelExists(ctx context.Context, pool *pgxpool.Pool, channelID uuid.UUID) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, "select exists(select 1 from channels where id = $1) as exists", channelID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

var ErrChannelNotFound = errors.New("channel not found")
