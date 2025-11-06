package replication

import (
	"context"
	"fmt"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/pg/dts/internal/wal"
)

// Subscriber WAL 订阅者
type Subscriber struct {
	conn     *pgconn.PgConn
	decoder  *wal.Decoder
	handler  *wal.Handler
	slotName string
}

// NewSubscriber 创建订阅者
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

// Close 关闭连接
func (s *Subscriber) Close() error {
	if s.conn != nil {
		return s.conn.Close(context.Background())
	}
	return nil
}

// StartReplication 开始复制
func (s *Subscriber) StartReplication(ctx context.Context, publicationName string) error {
	// 创建复制流
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

// ProcessReplicationStream 处理复制流
func (s *Subscriber) ProcessReplicationStream(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 接收消息
			msg, err := s.conn.ReceiveMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to receive message: %w", err)
			}

			// 处理消息
			switch v := msg.(type) {
			case *pgproto3.CopyData:
				if err := s.handleCopyData(ctx, v); err != nil {
					return err
				}
			case *pgproto3.NoticeResponse:
				// 处理通知消息
			case *pgproto3.ParameterStatus:
				// 处理参数状态
			default:
				// 其他类型的消息
			}
		}
	}
}

// handleCopyData 处理复制数据
func (s *Subscriber) handleCopyData(ctx context.Context, msg *pgproto3.CopyData) error {
	switch msg.Data[0] {
	case pglogrepl.PrimaryKeepaliveMessageByteID:
		// 处理 keepalive 消息
		pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
		if err != nil {
			return fmt.Errorf("failed to parse keepalive: %w", err)
		}

		if pkm.ServerWALEnd > pglogrepl.LSN(0) {
			// 发送确认
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
		// 处理 XLog 数据
		xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
		if err != nil {
			return fmt.Errorf("failed to parse xlog data: %w", err)
		}

		// 解析逻辑复制消息
		logicalMsg, err := pglogrepl.Parse(xld.WALData)
		if err != nil {
			return fmt.Errorf("failed to parse logical message: %w", err)
		}

		// 解码消息
		decodedMsg, err := s.decoder.Decode(logicalMsg)
		if err != nil {
			return fmt.Errorf("failed to decode message: %w", err)
		}

		// 处理消息
		if err := s.handler.Handle(ctx, decodedMsg); err != nil {
			return fmt.Errorf("failed to handle message: %w", err)
		}

		// 发送确认
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
