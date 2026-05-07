package cdn

import (
	"github.com/thatrajaryan/web-server/common"
)

type CdnBlock struct {
	CacheSize int
}

func (b *CdnBlock) Create(config map[string]interface{}) error {
	return nil
}

func (b *CdnBlock) Connect(target common.Block) error {
	return nil
}

func (b *CdnBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *CdnBlock) Delete() error {
	return nil
}

func (b *CdnBlock) Status() string {
	return "Active"
}

func (b *CdnBlock) Start() error {
	return nil
}

func (b *CdnBlock) Stop() error {
	return nil
}
