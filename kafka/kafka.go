package kafka

import (
	"fmt"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/thatrajaryan/web-server/common"
)

type KafkaBlock struct {
	BootstrapServers string
	Topic           string
	GroupID         string
	AutoOffsetReset string
	SecurityProtocol string
	Partitions      int
	ReplicationFactor int
	Producer        *ckafka.Producer
}

func (b *KafkaBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Kafka] Initializing Confluent Kafka block...")
	return b.Update(config)
}

func (b *KafkaBlock) Connect(target common.Block) error {
	fmt.Printf("[Kafka] Connecting to block. Topic: %s, Servers: %s\n", b.Topic, b.BootstrapServers)
	return nil
}

func (b *KafkaBlock) Update(config map[string]interface{}) error {
	if val, ok := config["bootstrap_servers"].(string); ok {
		b.BootstrapServers = val
	}
	if val, ok := config["topic"].(string); ok {
		b.Topic = val
	}
	if val, ok := config["group_id"].(string); ok {
		b.GroupID = val
	}
	if val, ok := config["auto_offset_reset"].(string); ok {
		b.AutoOffsetReset = val
	}
	if val, ok := config["security_protocol"].(string); ok {
		b.SecurityProtocol = val
	}
	if val, ok := config["partitions"].(float64); ok {
		b.Partitions = int(val)
	}
	if val, ok := config["replication_factor"].(float64); ok {
		b.ReplicationFactor = int(val)
	}

	fmt.Printf("[Kafka] Confluent Config: Servers=%s, Topic=%s, Group=%s, Security=%s\n",
		b.BootstrapServers, b.Topic, b.GroupID, b.SecurityProtocol)

	// In a real implementation, we would re-initialize the producer here if servers changed
	// configMap := &ckafka.ConfigMap{
	// 	"bootstrap.servers": b.BootstrapServers,
	// 	"group.id":          b.GroupID,
	// 	"auto.offset.reset": b.AutoOffsetReset,
	// 	"security.protocol": b.SecurityProtocol,
	// }
	// p, err := ckafka.NewProducer(configMap)
	// if err == nil { b.Producer = p }

	return nil
}

func (b *KafkaBlock) Delete() error {
	if b.Producer != nil {
		b.Producer.Close()
	}
	fmt.Printf("[Kafka] Cleaned up Confluent Kafka producer for topic %s\n", b.Topic)
	return nil
}