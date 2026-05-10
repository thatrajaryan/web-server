package database

import (
	"fmt"
	"github.com/thatrajaryan/web-server/common"
)

type DatabaseBlock struct {
	Type       string
	ConnString string
	StorageGB  int
}

func (b *DatabaseBlock) Create(config map[string]interface{}) error {
	return b.Update(config)
}

func (b *DatabaseBlock) Connect(target common.Block) error {
	// Implementation to connect to target blocks
	return nil
}

func (b *DatabaseBlock) Update(config map[string]interface{}) error {
	if val, ok := config["type"].(string); ok {
		b.Type = val
	}
	if val, ok := config["connection_string"].(string); ok {
		b.ConnString = val
	}
	if val, ok := config["storage_gb"].(float64); ok {
		b.StorageGB = int(val)
	}
	fmt.Printf("[Database] Configuration updated: Type=%s, Storage=%dGB\n", b.Type, b.StorageGB)
	return nil
}

func (b *DatabaseBlock) Delete() error {
	return nil
}
