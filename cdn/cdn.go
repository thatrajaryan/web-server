package cdn

import (
	"fmt"
	"github.com/hashicorp/golang-lru/v2"
	"github.com/thatrajaryan/web-server/common"
)

type CdnBlock struct {
	CacheSize   int
	TTL         int
	Compression bool
	Cache       *lru.Cache[string, interface{}]
}

func (b *CdnBlock) Create(config map[string]interface{}) error {
	fmt.Println("[CDN] Initializing Edge CDN...")
	return b.Update(config)
}

func (b *CdnBlock) Connect(target common.Block) error {
	fmt.Printf("[CDN] Origin server connected. TTL: %d\n", b.TTL)
	return nil
}

func (b *CdnBlock) Update(config map[string]interface{}) error {
	if val, ok := config["cache_size"].(float64); ok {
		b.CacheSize = int(val)
	}
	if val, ok := config["ttl"].(float64); ok {
		b.TTL = int(val)
	}
	if val, ok := config["compression"].(bool); ok {
		b.Compression = val
	}

	// Re-initialize LRU cache if size changed
	var err error
	b.Cache, err = lru.New[string, interface{}](b.CacheSize)
	if err != nil {
		return fmt.Errorf("failed to create LRU cache: %v", err)
	}

	fmt.Printf("[CDN] Configuration updated: CacheSize=%d entries, TTL=%ds, Compression=%v\n",
		b.CacheSize, b.TTL, b.Compression)
	return nil
}

func (b *CdnBlock) Delete() error {
	b.Cache = nil
	fmt.Println("[CDN] Cache purged and CDN block deleted")
	return nil
}
