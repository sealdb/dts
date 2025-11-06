package wal

import (
	"context"
	"fmt"
)

// Handler WAL 变更处理器
type Handler struct {
	tableMapping map[int]TableMapping // relationID -> table mapping
}

// TableMapping 表映射
type TableMapping struct {
	Schema     string
	TableName  string
	TargetName string // 目标表名（带后缀）
	Columns    []string
}

// NewHandler 创建处理器
func NewHandler() *Handler {
	return &Handler{
		tableMapping: make(map[int]TableMapping),
	}
}

// RegisterTable 注册表映射
func (h *Handler) RegisterTable(relationID int, schema, tableName, targetName string) {
	h.tableMapping[relationID] = TableMapping{
		Schema:     schema,
		TableName:  tableName,
		TargetName: targetName,
	}
}

// Handle 处理 WAL 消息
func (h *Handler) Handle(ctx context.Context, msg Message) error {
	switch v := msg.(type) {
	case *RelationMessage:
		// 关系消息，记录表映射关系
		cols := make([]string, len(v.Columns))
		for i, c := range v.Columns {
			cols[i] = c.Name
		}
		// 以 schema.tableName 作为 key 注册，TargetName 暂留，由上层注入时注册
		if m, ok := h.tableMapping[v.RelationID]; ok {
			m.Columns = cols
			h.tableMapping[v.RelationID] = m
		} else {
			h.tableMapping[v.RelationID] = TableMapping{
				Schema:     v.Namespace,
				TableName:  v.RelationName,
				TargetName: v.RelationName, // 默认同名，上层可带后缀覆盖
				Columns:    cols,
			}
		}
		return nil

	case *InsertMessage:
		return h.handleInsert(ctx, v)

	case *UpdateMessage:
		return h.handleUpdate(ctx, v)

	case *DeleteMessage:
		return h.handleDelete(ctx, v)

	case *TruncateMessage:
		return h.handleTruncate(ctx, v)

	case *BeginMessage:
		// 开始事务，可以在这里初始化事务上下文
		return nil

	case *CommitMessage:
		// 提交事务
		return nil

	default:
		return fmt.Errorf("unknown message type: %s", msg.Type())
	}
}

// handleInsert 处理插入
func (h *Handler) handleInsert(ctx context.Context, msg *InsertMessage) error {
	mapping, ok := h.tableMapping[msg.RelationID]
	if !ok {
		return fmt.Errorf("unknown relation ID: %d", msg.RelationID)
	}

	values := tupleToMap(mapping.Columns, msg.Tuple)
	_ = values
	// 实际执行应调用目标库执行，这里留给上层集成（TargetRepository.ApplyInsert）
	return nil
}

// handleUpdate 处理更新
func (h *Handler) handleUpdate(ctx context.Context, msg *UpdateMessage) error {
	mapping, ok := h.tableMapping[msg.RelationID]
	if !ok {
		return fmt.Errorf("unknown relation ID: %d", msg.RelationID)
	}

	oldVals := tupleToMap(mapping.Columns, msg.OldTuple)
	newVals := tupleToMap(mapping.Columns, msg.NewTuple)
	_, _ = oldVals, newVals
	return nil
}

// handleDelete 处理删除
func (h *Handler) handleDelete(ctx context.Context, msg *DeleteMessage) error {
	mapping, ok := h.tableMapping[msg.RelationID]
	if !ok {
		return fmt.Errorf("unknown relation ID: %d", msg.RelationID)
	}

	where := tupleToMap(mapping.Columns, msg.OldTuple)
	_ = where
	return nil
}

// handleTruncate 处理截断
func (h *Handler) handleTruncate(ctx context.Context, msg *TruncateMessage) error {
	// TODO: 处理截断操作（需要根据 RelationIDs 找到表并执行 TRUNCATE）
	return nil
}

// tupleToMap 将 Tuple 转换为 列名->值 的 map
func tupleToMap(columns []string, tuple *Tuple) map[string]interface{} {
	result := make(map[string]interface{})
	if tuple == nil {
		return result
	}
	for i := range tuple.Columns {
		if i >= len(columns) {
			break
		}
		col := tuple.Columns[i]
		name := columns[i]
		switch col.Kind {
		case 'n':
			result[name] = nil
		case 't':
			result[name] = string(col.Data)
		case 'u':
			// unchanged TOASTed value; skip
		default:
			result[name] = string(col.Data)
		}
	}
	return result
}
