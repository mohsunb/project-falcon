package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Channel struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Position          int       `json:"position"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
}

type ChannelCreationRequest struct {
	Name string `json:"name"`
}

type ChannelRepositionRequest struct {
	Position int `json:"position"`
}

func CreateChannel(ctx context.Context, conn *pgxpool.Pool, request ChannelCreationRequest) error {
	query := "insert into channels (id, name, position, creation_timestamp) values ($1, $2, $3, $4)"
	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate a random id for the channel: %w", err)
	}
	position, err := determineNextPosition(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to determine next position: %w", err)
	}
	if _, err := conn.Exec(ctx, query,
		id,
		request.Name,
		position,
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("failed to create the channel: %w", err)
	}
	return nil
}

func determineNextPosition(ctx context.Context, conn *pgxpool.Pool) (int, error) {
	var lastPosition int
	if err := conn.QueryRow(ctx, "select position from channels order by position desc limit 1").Scan(&lastPosition); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return -1, err
		}
		return 0, nil
	}
	return lastPosition + 1, nil
}

func GetAllChannels(ctx context.Context, conn *pgxpool.Pool) ([]Channel, error) {
	channels := make([]Channel, 0)
	rows, err := conn.Query(ctx, "select id, name, position, creation_timestamp from channels")
	if err != nil {
		return nil, fmt.Errorf("failed to get all channels: %w", err)
	}

	for rows.Next() {
		var channel Channel
		rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.Position,
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

func RepositionChannel(ctx context.Context, conn *pgxpool.Pool, id uuid.UUID, request ChannelRepositionRequest) error {
	channelExists, err := ChannelExists(ctx, conn, id)
	if err != nil {
		return fmt.Errorf("failed to check if the channel exists: %w", err)
	}
	if !channelExists {
		return ErrChannelNotFound
	}

	if err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) (err error) {
		_, err = tx.Exec(ctx, "select id from channels for update")
		_, err = tx.Exec(ctx, "update channels set position = $1 where id = $2", request.Position, id)
		_, err = tx.Exec(ctx, "update channels set position = position + 1 where position >= $1 and id != $2", request.Position, id)
		return
	}); err != nil {
		return fmt.Errorf("failed to reposition channel: %w", err)
	}

	return nil
}
