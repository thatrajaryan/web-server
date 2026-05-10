package server

import (
	"fmt"
	"github.com/thatrajaryan/web-server/common"
)

type ServerBlock struct {
	IP        string
	Port      int
	CPU       int
	Memory    string
}

func (b *ServerBlock) Create(config map[string]interface{}) error {
	return b.Update(config)
}

func (b *ServerBlock) Connect(target common.Block) error {
	// Implementation to connect to target blocks
	return nil
}

func (b *ServerBlock) Update(config map[string]interface{}) error {
	if val, ok := config["port"].(float64); ok {
		b.Port = int(val)
	}
	if val, ok := config["cpu"].(float64); ok {
		b.CPU = int(val)
	}
	if val, ok := config["memory"].(string); ok {
		b.Memory = val
	}
	fmt.Printf("[Server] Configuration updated: Port=%d, CPU=%d cores, Memory=%s\n", b.Port, b.CPU, b.Memory)
	return nil
}

func (b *ServerBlock) Delete() error {
	return nil
}