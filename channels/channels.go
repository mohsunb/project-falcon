package channels

import (
	"context"
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
		return fmt.Errorf("failed to generate a random id for the channel: %v", err)
	}
	if _, err := conn.Exec(ctx, query,
		id,
		request.Name,
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("failed to create the channel: %v", err)
	}
	return nil
}

func GetAllChannels(ctx context.Context, conn *pgxpool.Pool) ([]Channel, error) {
	channels := make([]Channel, 0)
	rows, err := conn.Query(ctx, "select id, name, creation_timestamp from channels")
	if err != nil {
		return nil, fmt.Errorf("failed to get all channels: %v", err)
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
		return channels, fmt.Errorf("failed to read all channels: %v", err)
	}

	return channels, nil
}
