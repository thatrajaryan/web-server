package api

// Database is a generic interface for database operations.
type Database interface {
	Connect() error
	Disconnect() error
	Query(query string, args ...interface{}) (interface{}, error)
}
