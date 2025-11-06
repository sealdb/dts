package wal

import (
	"fmt"

	"github.com/jackc/pglogrepl"
)

// Decoder WAL 解码器
type Decoder struct {
	plugin string
}

// NewDecoder 创建解码器
func NewDecoder(plugin string) *Decoder {
	if plugin == "" {
		plugin = "pgoutput" // 默认使用 pgoutput
	}
	return &Decoder{plugin: plugin}
}

// Decode 解码 WAL 消息
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

// convertColumns 转换列信息
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

// convertTuple 转换元组
func convertTuple(tuple *pglogrepl.TupleData) *Tuple {
	if tuple == nil {
		return nil
	}

	result := &Tuple{
		Columns: make([]TupleColumn, len(tuple.Columns)),
	}

	for i, col := range tuple.Columns {
		// 注意：新版本的 pglogrepl 中，TupleDataColumn 可能没有 Kind 字段
		// 使用 DataType 来判断类型
		result.Columns[i] = TupleColumn{
			Kind:     0, // 如果 Kind 不存在，使用 0 或根据 DataType 推断
			DataType: int(col.DataType),
			Length:   int(col.Length),
			Data:     col.Data,
		}
	}

	return result
}

// convertRelationIDs 转换关系ID列表
func convertRelationIDs(ids []uint32) []int {
	result := make([]int, len(ids))
	for i, id := range ids {
		result[i] = int(id)
	}
	return result
}
