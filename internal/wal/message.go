package wal

import "time"

// Message represents WAL message interface
type Message interface {
	Type() string
}

// RelationMessage represents relation message
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

// Column represents column information
type Column struct {
	Flags        int
	Name         string
	DataTypeOID  int
	TypeModifier int
}

// InsertMessage represents insert message
type InsertMessage struct {
	RelationID int
	Tuple      *Tuple
}

func (m *InsertMessage) Type() string {
	return "insert"
}

// UpdateMessage represents update message
type UpdateMessage struct {
	RelationID int
	OldTuple   *Tuple
	NewTuple   *Tuple
}

func (m *UpdateMessage) Type() string {
	return "update"
}

// DeleteMessage represents delete message
type DeleteMessage struct {
	RelationID int
	OldTuple   *Tuple
}

func (m *DeleteMessage) Type() string {
	return "delete"
}

// TruncateMessage represents truncate message
type TruncateMessage struct {
	RelationIDs []int
}

func (m *TruncateMessage) Type() string {
	return "truncate"
}

// BeginMessage represents begin transaction message
type BeginMessage struct {
	FinalLSN  string
	Timestamp time.Time
	XID       int
}

func (m *BeginMessage) Type() string {
	return "begin"
}

// CommitMessage represents commit transaction message
type CommitMessage struct {
	Flags             int
	LSN               string
	TransactionEndLSN string
	Timestamp         time.Time
}

func (m *CommitMessage) Type() string {
	return "commit"
}

// Tuple represents a tuple
type Tuple struct {
	Columns []TupleColumn
}

// TupleColumn represents a tuple column
type TupleColumn struct {
	// Kind follows pgoutput: 'n' = null, 't' = text, 'u' = unchanged toast
	Kind     byte
	DataType int
	Length   int
	Data     []byte
}
