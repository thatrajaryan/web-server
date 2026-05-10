// first point of contact for any api. Performs TLS handshakes and gives it forward to load balancer

package api_gateway

import (
	"fmt"
	"github.com/thatrajaryan/web-server/common"
)

type ApiGatewayBlock struct {
	Port          int
	RateLimit     int
	LBAlgorithm   string
	CertPath      string
	KeyPath       string
}

func (b *ApiGatewayBlock) Create(config map[string]interface{}) error {
	return b.Update(config)
}

func (b *ApiGatewayBlock) Connect(target common.Block) error {
	// Implementation to connect to target downstream blocks
	return nil
}

func (b *ApiGatewayBlock) Update(config map[string]interface{}) error {
	if val, ok := config["port"].(float64); ok {
		b.Port = int(val)
	}
	if val, ok := config["rate_limit"].(float64); ok {
		b.RateLimit = int(val)
	}
	if val, ok := config["lb_algo"].(string); ok {
		b.LBAlgorithm = val
	}
	fmt.Printf("[API Gateway] Configuration updated: Port=%d, RateLimit=%d, LBAlgo=%s\n", b.Port, b.RateLimit, b.LBAlgorithm)
	return nil
}

func (b *ApiGatewayBlock) Delete() error {
	return nil
}