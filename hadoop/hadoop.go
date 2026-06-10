package hadoop

import (
	"fmt"
	"github.com/thatrajaryan/web-server/common"
)

type HadoopBlock struct {
	FsDefaultName                  string
	DfsReplication                 int
	DfsNamenodeNameDir             string
	DfsDatanodeDataDir             string
	YarnResourcemanagerHostname    string
	YarnNodemanagerResourceMemoryMb int
}

func (b *HadoopBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Hadoop] Initializing Hadoop cluster block...")
	return b.Update(config)
}

func (b *HadoopBlock) Connect(target common.Block) error {
	fmt.Printf("[Hadoop] Connecting Hadoop cluster to target block\n")
	return nil
}

func (b *HadoopBlock) Update(config map[string]interface{}) error {
	if val, ok := config["fs_default_name"].(string); ok {
		b.FsDefaultName = val
	}
	if val, ok := config["dfs_replication"].(float64); ok {
		b.DfsReplication = int(val)
	}
	if val, ok := config["dfs_namenode_name_dir"].(string); ok {
		b.DfsNamenodeNameDir = val
	}
	if val, ok := config["dfs_datanode_data_dir"].(string); ok {
		b.DfsDatanodeDataDir = val
	}
	if val, ok := config["yarn_resourcemanager_hostname"].(string); ok {
		b.YarnResourcemanagerHostname = val
	}
	if val, ok := config["yarn_nodemanager_resource_memory_mb"].(float64); ok {
		b.YarnNodemanagerResourceMemoryMb = int(val)
	}

	fmt.Printf("[Hadoop] Hadoop Configured: FS=%s, Replication=%d, ResourceManager=%s\n",
		b.FsDefaultName, b.DfsReplication, b.YarnResourcemanagerHostname)
	return nil
}

func (b *HadoopBlock) Delete() error {
	fmt.Println("[Hadoop] Hadoop cluster decommissioned")
	return nil
}
