package api

import (
	"database/sql"
)

// Database is a generic interface for database operations.
type Database interface {
	Connect() error
	Disconnect() error
	Query(query string, args ...interface{}) (interface{}, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Migrate() error
	Begin() (*sql.Tx, error)
}
