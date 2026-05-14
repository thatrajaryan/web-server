package rate_limiter

import (
	"fmt"
	"golang.org/x/time/rate"
	"github.com/thatrajaryan/web-server/common"
)

type RateLimiterBlock struct {
	RPS      float64
	Burst    int
	Strategy string
	Limiter  *rate.Limiter
}

func (b *RateLimiterBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Rate Limiter] Initializing Token-Bucket Limiter...")
	return b.Update(config)
}

func (b *RateLimiterBlock) Connect(target common.Block) error {
	fmt.Printf("[Rate Limiter] Limiting connections to target. RPS: %.2f\n", b.RPS)
	return nil
}

func (b *RateLimiterBlock) Update(config map[string]interface{}) error {
	if val, ok := config["rate"].(float64); ok {
		b.RPS = val
	}
	if val, ok := config["burst"].(float64); ok {
		b.Burst = int(val)
	}
	if val, ok := config["strategy"].(string); ok {
		b.Strategy = val
	}

	// Update standard library limiter
	b.Limiter = rate.NewLimiter(rate.Limit(b.RPS), b.Burst)

	fmt.Printf("[Rate Limiter] Configuration updated: RPS=%.2f, Burst=%d, Strategy=%s\n",
		b.RPS, b.Burst, b.Strategy)
	return nil
}

func (b *RateLimiterBlock) Delete() error {
	b.Limiter = nil
	fmt.Println("[Rate Limiter] Limiter deleted")
	return nil
}
