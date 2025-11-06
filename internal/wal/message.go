package wal

import "time"

// Message WAL 消息接口
type Message interface {
	Type() string
}

// RelationMessage 关系消息
type RelationMessage struct {
	RelationID      int
	Namespace       string
	RelationName    string
	ReplicaIdentity string
	Columns         []Column
}

func (m *RelationMessage) Type() string {
	return "relation"
}

// Column 列信息
type Column struct {
	Flags        int
	Name         string
	DataTypeOID  int
	TypeModifier int
}

// InsertMessage 插入消息
type InsertMessage struct {
	RelationID int
	Tuple      *Tuple
}

func (m *InsertMessage) Type() string {
	return "insert"
}

// UpdateMessage 更新消息
type UpdateMessage struct {
	RelationID int
	OldTuple   *Tuple
	NewTuple   *Tuple
}

func (m *UpdateMessage) Type() string {
	return "update"
}

// DeleteMessage 删除消息
type DeleteMessage struct {
	RelationID int
	OldTuple   *Tuple
}

func (m *DeleteMessage) Type() string {
	return "delete"
}

// TruncateMessage 截断消息
type TruncateMessage struct {
	RelationIDs []int
}

func (m *TruncateMessage) Type() string {
	return "truncate"
}

// BeginMessage 开始事务消息
type BeginMessage struct {
	FinalLSN  string
	Timestamp time.Time
	XID       int
}

func (m *BeginMessage) Type() string {
	return "begin"
}

// CommitMessage 提交事务消息
type CommitMessage struct {
	Flags             int
	LSN               string
	TransactionEndLSN string
	Timestamp         time.Time
}

func (m *CommitMessage) Type() string {
	return "commit"
}

// Tuple 元组
type Tuple struct {
	Columns []TupleColumn
}

// TupleColumn 元组列
type TupleColumn struct {
	// Kind follows pgoutput: 'n' = null, 't' = text, 'u' = unchanged toast
	Kind     byte
	DataType int
	Length   int
	Data     []byte
}
