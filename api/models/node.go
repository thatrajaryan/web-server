package models

import "time"

// Node represents an architectural block on the canvas.
type Node struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	Type      string                 `json:"type"` // api_gateway, code, kafka, database, server, cdn
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
