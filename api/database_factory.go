package api

import (
	"fmt"
)

// NewDatabase returns a new database instance based on the provided strategy.
func NewDatabase(strategy string) (Database, error) {
	switch strategy {
	case "postgres":
		return &PostgresStrategy{}, nil
	default:
		return nil, fmt.Errorf("unsupported database strategy: %s", strategy)
	}
}
