package models

import "time"

// Connection represents a link between two architectural blocks.
type Connection struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	FromNodeID string    `json:"from_node_id"`
	ToNodeID   string    `json:"to_node_id"`
	HookCode   string    `json:"hook_code"`
	CreatedAt  time.Time `json:"created_at"`
}
