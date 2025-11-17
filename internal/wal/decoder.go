package wal

import (
	"fmt"

	"github.com/jackc/pglogrepl"
)

// Decoder decodes WAL messages
type Decoder struct {
	plugin string
}

// NewDecoder creates a decoder
func NewDecoder(plugin string) *Decoder {
	if plugin == "" {
		plugin = "pgoutput" // Default to pgoutput
	}
	return &Decoder{plugin: plugin}
}

// Decode decodes WAL messages
func (d *Decoder) Decode(msg pglogrepl.Message) (Message, error) {
	switch v := msg.(type) {
	case *pglogrepl.RelationMessage:
		return &RelationMessage{
			RelationID:      int(v.RelationID),
			Namespace:       v.Namespace,
			RelationName:    v.RelationName,
			ReplicaIdentity: string(v.ReplicaIdentity),
			Columns:         convertColumns(v.Columns),
		}, nil

	case *pglogrepl.InsertMessage:
		return &InsertMessage{
			RelationID: int(v.RelationID),
			Tuple:      convertTuple(v.Tuple),
		}, nil

	case *pglogrepl.UpdateMessage:
		return &UpdateMessage{
			RelationID: int(v.RelationID),
			OldTuple:   convertTuple(v.OldTuple),
			NewTuple:   convertTuple(v.NewTuple),
		}, nil

	case *pglogrepl.DeleteMessage:
		return &DeleteMessage{
			RelationID: int(v.RelationID),
			OldTuple:   convertTuple(v.OldTuple),
		}, nil

	case *pglogrepl.TruncateMessage:
		return &TruncateMessage{
			RelationIDs: convertRelationIDs(v.RelationIDs),
		}, nil

	case *pglogrepl.BeginMessage:
		return &BeginMessage{
			FinalLSN:  v.FinalLSN.String(),
			Timestamp: v.CommitTime,
			XID:       int(v.Xid),
		}, nil

	case *pglogrepl.CommitMessage:
		return &CommitMessage{
			Flags:             int(v.Flags),
			LSN:               v.CommitLSN.String(),
			TransactionEndLSN: v.TransactionEndLSN.String(),
			Timestamp:         v.CommitTime,
		}, nil

	default:
		return nil, fmt.Errorf("unknown message type: %T", v)
	}
}

// convertColumns converts column information
func convertColumns(cols []*pglogrepl.RelationMessageColumn) []Column {
	result := make([]Column, len(cols))
	for i, col := range cols {
		result[i] = Column{
			Flags:        int(col.Flags),
			Name:         col.Name,
			DataTypeOID:  int(col.DataType),
			TypeModifier: int(col.TypeModifier),
		}
	}
	return result
}

// convertTuple converts tuple
func convertTuple(tuple *pglogrepl.TupleData) *Tuple {
	if tuple == nil {
		return nil
	}

	result := &Tuple{
		Columns: make([]TupleColumn, len(tuple.Columns)),
	}

	for i, col := range tuple.Columns {
		// Note: In newer versions of pglogrepl, TupleDataColumn may not have Kind field
		// Use DataType to determine type
		result.Columns[i] = TupleColumn{
			Kind:     0, // If Kind doesn't exist, use 0 or infer from DataType
			DataType: int(col.DataType),
			Length:   int(col.Length),
			Data:     col.Data,
		}
	}

	return result
}

// convertRelationIDs converts relation ID list
func convertRelationIDs(ids []uint32) []int {
	result := make([]int, len(ids))
	for i, id := range ids {
		result[i] = int(id)
	}
	return result
}
