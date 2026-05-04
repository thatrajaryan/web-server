package rate_limiter

import (
	"log"
)

type RateLimiter interface {
	Count(addr String)
}

type BucketStrategy struct {
	bucket map[int]int
}

func (limiter *BucketStrategy) Count(addr String) bool {

}