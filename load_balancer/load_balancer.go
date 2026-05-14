package load_balancer

import (
	"fmt"
	"sync"
	"github.com/thatrajaryan/web-server/common"
)

type LoadBalancerBlock struct {
	Algorithm            string
	HealthCheckInterval int
	MaxRetries          int
	mu                  sync.Mutex
	targets             []common.Block
	currentIndex        int
}

func (b *LoadBalancerBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Load Balancer] Initializing Load Balancer...")
	b.targets = []common.Block{}
	return b.Update(config)
}

func (b *LoadBalancerBlock) Connect(target common.Block) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.targets = append(b.targets, target)
	fmt.Printf("[Load Balancer] Registered new target. Total targets: %d\n", len(b.targets))
	return nil
}

func (b *LoadBalancerBlock) Update(config map[string]interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if val, ok := config["algorithm"].(string); ok {
		b.Algorithm = val
	}
	if val, ok := config["health_check_interval"].(float64); ok {
		b.HealthCheckInterval = int(val)
	}
	if val, ok := config["max_retries"].(float64); ok {
		b.MaxRetries = int(val)
	}

	fmt.Printf("[Load Balancer] Configuration updated: Algorithm=%s, HealthCheck=%ds, MaxRetries=%d\n",
		b.Algorithm, b.HealthCheckInterval, b.MaxRetries)
	return nil
}

func (b *LoadBalancerBlock) Delete() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.targets = nil
	fmt.Println("[Load Balancer] All targets deregistered and LB deleted")
	return nil
}

// NextTarget returns the next block according to the selected algorithm
func (b *LoadBalancerBlock) NextTarget() common.Block {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if len(b.targets) == 0 {
		return nil
	}

	switch b.Algorithm {
	case "round-robin":
		target := b.targets[b.currentIndex]
		b.currentIndex = (b.currentIndex + 1) % len(b.targets)
		return target
	case "random":
		// Simplified random
		return b.targets[0] 
	default:
		return b.targets[0]
	}
}
