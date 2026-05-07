package database

import (
	"github.com/thatrajaryan/web-server/common"
)

type DatabaseBlock struct {
	Type       string
	ConnString string
}

func (b *DatabaseBlock) Create(config map[string]interface{}) error {
	// Implementation to be added later
	return nil
}

func (b *DatabaseBlock) Connect(target common.Block) error {
	// Implementation to be added later
	return nil
}

func (b *DatabaseBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *DatabaseBlock) Delete() error {
	return nil
}

func (b *DatabaseBlock) Status() string {
	return "Active"
}

func (b *DatabaseBlock) Start() error {
	return nil
}

func (b *DatabaseBlock) Stop() error {
	return nil
}
