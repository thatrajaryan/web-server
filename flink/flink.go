package flink

import (
	"fmt"
	"github.com/thatrajaryan/web-server/common"
)

type FlinkBlock struct {
	JobManagerRpcAddress string
	JobManagerRpcPort    int
	JobManagerHeapSize   string
	TaskManagerHeapSize  string
	TaskManagerSlots     int
	ParallelismDefault   int
	StateBackend         string
}

func (b *FlinkBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Flink] Initializing Flink cluster block...")
	return b.Update(config)
}

func (b *FlinkBlock) Connect(target common.Block) error {
	fmt.Printf("[Flink] Connecting Flink cluster to target block\n")
	return nil
}

func (b *FlinkBlock) Update(config map[string]interface{}) error {
	if val, ok := config["jobmanager_rpc_address"].(string); ok {
		b.JobManagerRpcAddress = val
	}
	if val, ok := config["jobmanager_rpc_port"].(float64); ok {
		b.JobManagerRpcPort = int(val)
	}
	if val, ok := config["jobmanager_heap_size"].(string); ok {
		b.JobManagerHeapSize = val
	}
	if val, ok := config["taskmanager_heap_size"].(string); ok {
		b.TaskManagerHeapSize = val
	}
	if val, ok := config["taskmanager_slots"].(float64); ok {
		b.TaskManagerSlots = int(val)
	}
	if val, ok := config["parallelism_default"].(float64); ok {
		b.ParallelismDefault = int(val)
	}
	if val, ok := config["state_backend"].(string); ok {
		b.StateBackend = val
	}

	fmt.Printf("[Flink] Flink Configured: JobManager=%s:%d, StateBackend=%s\n",
		b.JobManagerRpcAddress, b.JobManagerRpcPort, b.StateBackend)
	return nil
}

func (b *FlinkBlock) Delete() error {
	fmt.Println("[Flink] Flink cluster decommissioned")
	return nil
}
