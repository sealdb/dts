package database

import "errors"

var (
	// ErrUnsupportedDatabaseType indicates unsupported database type
	ErrUnsupportedDatabaseType = errors.New("unsupported database type")
)

