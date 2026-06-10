package spark

import (
	"fmt"
	"github.com/thatrajaryan/web-server/common"
)

type SparkBlock struct {
	MasterURL         string
	DriverMemory      string
	ExecutorMemory    string
	ExecutorCores     int
	DeployMode        string
	DynamicAllocation bool
}

func (b *SparkBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Spark] Initializing Spark cluster block...")
	return b.Update(config)
}

func (b *SparkBlock) Connect(target common.Block) error {
	fmt.Printf("[Spark] Connecting Spark cluster to target block\n")
	return nil
}

func (b *SparkBlock) Update(config map[string]interface{}) error {
	if val, ok := config["master_url"].(string); ok {
		b.MasterURL = val
	}
	if val, ok := config["driver_memory"].(string); ok {
		b.DriverMemory = val
	}
	if val, ok := config["executor_memory"].(string); ok {
		b.ExecutorMemory = val
	}
	if val, ok := config["executor_cores"].(float64); ok {
		b.ExecutorCores = int(val)
	}
	if val, ok := config["deploy_mode"].(string); ok {
		b.DeployMode = val
	}
	if val, ok := config["dynamic_allocation"].(bool); ok {
		b.DynamicAllocation = val
	}

	fmt.Printf("[Spark] Spark Configured: Master=%s, DeployMode=%s, DynamicAllocation=%t\n",
		b.MasterURL, b.DeployMode, b.DynamicAllocation)
	return nil
}

func (b *SparkBlock) Delete() error {
	fmt.Println("[Spark] Spark cluster decommissioned")
	return nil
}
