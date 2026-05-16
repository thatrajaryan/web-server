package api

import (
	"database/sql"
	"github.com/thatrajaryan/web-server/api/models"
)

// Database is a generic interface for database operations.
type Database interface {
	Connect() error
	Disconnect() error
	Query(query string, args ...interface{}) (interface{}, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Migrate() error
	Begin() (*sql.Tx, error)
	GetNodesByProject(projectID string) ([]models.Node, error)
	GetConnectionsByProject(projectID string) ([]models.Connection, error)
	GetProject(id string) (*models.Project, error)
}
