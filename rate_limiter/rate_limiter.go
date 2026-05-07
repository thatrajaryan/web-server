package rate_limiter

import (
	"sync"

	"github.com/thatrajaryan/web-server/common"
)

type RateLimiter interface {
	Allow(addr string) bool
}

type RateLimiterBlock struct {
	Rate     int64
	Capacity int
}

func (b *RateLimiterBlock) Create(config map[string]interface{}) error {
	// Implementation to be added later
	return nil
}

func (b *RateLimiterBlock) Connect(target common.Block) error {
	// Implementation to be added later
	return nil
}

func (b *RateLimiterBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *RateLimiterBlock) Delete() error {
	return nil
}

func (b *RateLimiterBlock) Status() string {
	return "Active"
}

func (b *RateLimiterBlock) Start() error {
	return nil
}

func (b *RateLimiterBlock) Stop() error {
	return nil
}

type Bucket struct {
	Lock     sync.Mutex
	LastFill int64
	Count    int
}

type BucketStrategy struct {
	Rate     int64
	Capacity int
	Bucket   map[int]Bucket
}

func (limiter *BucketStrategy) Allow(addr string) bool {
	// Implementation...
	return true
}
