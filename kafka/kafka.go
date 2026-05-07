package kafka

import (
	"github.com/thatrajaryan/web-server/common"
)

type KafkaBlock struct {
	Brokers []string
	Topic   string
}

func (b *KafkaBlock) Create(config map[string]interface{}) error {
	// Implementation to be added later
	return nil
}

func (b *KafkaBlock) Connect(target common.Block) error {
	// Implementation to be added later
	return nil
}

func (b *KafkaBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *KafkaBlock) Delete() error {
	return nil
}

func (b *KafkaBlock) Status() string {
	return "Active"
}

func (b *KafkaBlock) Start() error {
	return nil
}

func (b *KafkaBlock) Stop() error {
	return nil
}