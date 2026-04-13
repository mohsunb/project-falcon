package channels

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func CreateChannel(ctx context.Context, db *pgxpool.Pool, request ChannelCreationRequest) error {
	query := "insert into channels (id, name, position, creation_timestamp) values ($1, $2, $3, $4)"
	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate a random id for the channel: %w", err)
	}
	position, err := determineNextPosition(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to determine next position: %w", err)
	}
	insertFunc := func(position int) (pgconn.CommandTag, error) {
		return db.Exec(ctx, query,
			id,
			request.Name,
			position,
			time.Now().UTC(),
		)
	}

	if _, err := insertFunc(position); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			for i := 0; i < 2; i++ {
				time.Sleep(time.Second)
				position, err = determineNextPosition(ctx, db)
				if err != nil {
					return fmt.Errorf("failed to determine next position: %w", err)
				}
				if _, err = insertFunc(position); err == nil {
					break
				}
			}
			if err != nil {
				return ErrTooManyRequests
			}
		} else {
			return fmt.Errorf("failed to create the channel: %w", err)
		}
	}
	return nil
}

var ErrTooManyRequests = errors.New("too many channel creation requests")

func determineNextPosition(ctx context.Context, db *pgxpool.Pool) (int, error) {
	var lastPosition int
	if err := db.QueryRow(ctx, "select position from channels order by position desc limit 1").Scan(&lastPosition); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return -1, err
		}
		return 0, nil
	}
	return lastPosition + 1, nil
}

func GetAllChannels(ctx context.Context, db *pgxpool.Pool) ([]Channel, error) {
	channels := make([]Channel, 0)
	rows, err := db.Query(ctx, "select id, name, position, creation_timestamp from channels order by position")
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

func ChannelExists(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (bool, error) {
	var exists bool
	if err := db.QueryRow(ctx, "select exists(select 1 from channels where id = $1) as exists", id).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

var ErrChannelNotFound = errors.New("channel not found")

func RepositionChannel(ctx context.Context, db *pgxpool.Pool, logger *slog.Logger, id uuid.UUID, request ChannelRepositionRequest) error {
	channel, err := getChannel(ctx, db, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrChannelNotFound
		} else {
			return fmt.Errorf("failed to find the channel: %w", err)
		}
	}

	if channel.Position == request.Position {
		logger.WarnContext(ctx, "channel is already in the desired position; skipping repositioning...")
		return nil
	}

	if err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) (err error) {
		_, err = tx.Exec(ctx, "set constraints all deferred")
		_, err = tx.Exec(ctx, "select id from channels for update")
		_, err = tx.Exec(ctx, "update channels set position = $1 where id = $2", request.Position, id)

		oldPosition := channel.Position
		newPosition := request.Position
		if newPosition > oldPosition {
			_, err = tx.Exec(ctx, "update channels set position = position - 1 where position <= $1 and position > $2 and id != $3", newPosition, oldPosition, id)
		} else {
			_, err = tx.Exec(ctx, "update channels set position = position + 1 where position >= $1 and position < $2 and id != $3", newPosition, oldPosition, id)
		}
		return
	}); err != nil {
		return fmt.Errorf("failed to reposition channel: %w", err)
	}

	return nil
}

func getChannel(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (Channel, error) {
	var channel Channel
	err := db.QueryRow(ctx, "select id, name, position, creation_timestamp from channels where id = $1", id).Scan(
		&channel.ID,
		&channel.Name,
		&channel.Position,
		&channel.CreationTimestamp,
	)
	return channel, err
}
