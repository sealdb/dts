package replication

import (
	"context"
	"fmt"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/pg/dts/internal/wal"
)

// Subscriber is a WAL subscriber
type Subscriber struct {
	conn     *pgconn.PgConn
	decoder  *wal.Decoder
	handler  *wal.Handler
	slotName string
}

// NewSubscriber creates a subscriber
func NewSubscriber(connString, slotName string) (*Subscriber, error) {
	conn, err := pgconn.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Subscriber{
		conn:     conn,
		decoder:  wal.NewDecoder("pgoutput"),
		handler:  wal.NewHandler(),
		slotName: slotName,
	}, nil
}

// Close closes the connection
func (s *Subscriber) Close() error {
	if s.conn != nil {
		return s.conn.Close(context.Background())
	}
	return nil
}

// StartReplication starts replication
func (s *Subscriber) StartReplication(ctx context.Context, publicationName string) error {
	// Create replication stream
	pluginArgs := []string{
		"proto_version", "1",
		"publication_names", publicationName,
	}

	err := pglogrepl.StartReplication(
		ctx,
		s.conn,
		s.slotName,
		pglogrepl.LSN(0),
		pglogrepl.StartReplicationOptions{PluginArgs: pluginArgs},
	)

	if err != nil {
		return fmt.Errorf("failed to start replication: %w", err)
	}

	return nil
}

// ProcessReplicationStream processes replication stream
func (s *Subscriber) ProcessReplicationStream(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Receive message
			msg, err := s.conn.ReceiveMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to receive message: %w", err)
			}

			// Process message
			switch v := msg.(type) {
			case *pgproto3.CopyData:
				if err := s.handleCopyData(ctx, v); err != nil {
					return err
				}
			case *pgproto3.NoticeResponse:
				// Handle notice message
			case *pgproto3.ParameterStatus:
				// Handle parameter status
			default:
				// Other message types
			}
		}
	}
}

// handleCopyData handles replication data
func (s *Subscriber) handleCopyData(ctx context.Context, msg *pgproto3.CopyData) error {
	switch msg.Data[0] {
	case pglogrepl.PrimaryKeepaliveMessageByteID:
		// Handle keepalive message
		pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
		if err != nil {
			return fmt.Errorf("failed to parse keepalive: %w", err)
		}

		if pkm.ServerWALEnd > pglogrepl.LSN(0) {
			// Send acknowledgment
			err = pglogrepl.SendStandbyStatusUpdate(
				ctx,
				s.conn,
				pglogrepl.StandbyStatusUpdate{
					WALWritePosition: pkm.ServerWALEnd,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to send status update: %w", err)
			}
		}

	case pglogrepl.XLogDataByteID:
		// Handle XLog data
		xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
		if err != nil {
			return fmt.Errorf("failed to parse xlog data: %w", err)
		}

		// Parse logical replication message
		logicalMsg, err := pglogrepl.Parse(xld.WALData)
		if err != nil {
			return fmt.Errorf("failed to parse logical message: %w", err)
		}

		// Decode message
		decodedMsg, err := s.decoder.Decode(logicalMsg)
		if err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}

		// Handle message
		if err := s.handler.Handle(ctx, decodedMsg); err != nil {
			return fmt.Errorf("failed to handle message: %w", err)
		}

		// Send acknowledgment
		err = pglogrepl.SendStandbyStatusUpdate(
			ctx,
			s.conn,
			pglogrepl.StandbyStatusUpdate{
				WALWritePosition: xld.WALStart + pglogrepl.LSN(len(xld.WALData)),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to send status update: %w", err)
		}
	}

	return nil
}
