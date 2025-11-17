package wal

import (
	"context"
	"fmt"
)

// Handler handles WAL changes
type Handler struct {
	tableMapping map[int]TableMapping // relationID -> table mapping
}

// TableMapping represents table mapping
type TableMapping struct {
	Schema     string
	TableName  string
	TargetName string // Target table name (with suffix)
	Columns    []string
}

// NewHandler creates a handler
func NewHandler() *Handler {
	return &Handler{
		tableMapping: make(map[int]TableMapping),
	}
}

// RegisterTable registers table mapping
func (h *Handler) RegisterTable(relationID int, schema, tableName, targetName string) {
	h.tableMapping[relationID] = TableMapping{
		Schema:     schema,
		TableName:  tableName,
		TargetName: targetName,
	}
}

// Handle processes WAL messages
func (h *Handler) Handle(ctx context.Context, msg Message) error {
	switch v := msg.(type) {
	case *RelationMessage:
		// Relation message, record table mapping
		cols := make([]string, len(v.Columns))
		for i, c := range v.Columns {
			cols[i] = c.Name
		}
		// Register with schema.tableName as key, TargetName reserved, will be registered when injected by upper layer
		if m, ok := h.tableMapping[v.RelationID]; ok {
			m.Columns = cols
			h.tableMapping[v.RelationID] = m
		} else {
			h.tableMapping[v.RelationID] = TableMapping{
				Schema:     v.Namespace,
				TableName:  v.RelationName,
				TargetName: v.RelationName, // Default same name, upper layer can override with suffix
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
		// Begin transaction, can initialize transaction context here
		return nil

	case *CommitMessage:
		// Commit transaction
		return nil

	default:
		return fmt.Errorf("unknown message type: %s", msg.Type())
	}
}

// handleInsert handles insert
func (h *Handler) handleInsert(ctx context.Context, msg *InsertMessage) error {
	mapping, ok := h.tableMapping[msg.RelationID]
	if !ok {
		return fmt.Errorf("unknown relation ID: %d", msg.RelationID)
	}

	values := tupleToMap(mapping.Columns, msg.Tuple)
	_ = values
	// Actual execution should call target database, left for upper layer integration (TargetRepository.ApplyInsert)
	return nil
}

// handleUpdate handles update
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

// handleDelete handles delete
func (h *Handler) handleDelete(ctx context.Context, msg *DeleteMessage) error {
	mapping, ok := h.tableMapping[msg.RelationID]
	if !ok {
		return fmt.Errorf("unknown relation ID: %d", msg.RelationID)
	}

	where := tupleToMap(mapping.Columns, msg.OldTuple)
	_ = where
	return nil
}

// handleTruncate handles truncate
func (h *Handler) handleTruncate(ctx context.Context, msg *TruncateMessage) error {
	// TODO: Handle truncate operation (need to find tables by RelationIDs and execute TRUNCATE)
	return nil
}

// tupleToMap converts Tuple to a map of column name -> value
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
