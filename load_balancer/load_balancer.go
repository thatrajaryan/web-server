package load_balancer

import (
	"log"
	"github.com/thatrajaryan/web-server/common"
)

type LoadBalancer struct {
	Servers map[int]common.Server
	ServerList []int
	Pointer int
	ServerPointer int
}

type LoadBalancerBlock struct {
	Algorithm string
}

func (b *LoadBalancerBlock) Create(config map[string]interface{}) error {
	// Implementation to be added later
	return nil
}

func (b *LoadBalancerBlock) Connect(target common.Block) error {
	// Implementation to be added later
	return nil
}

func (b *LoadBalancerBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *LoadBalancerBlock) Delete() error {
	return nil
}

func (b *LoadBalancerBlock) Status() string {
	return "Active"
}

func (b *LoadBalancerBlock) Start() error {
	return nil
}

func (b *LoadBalancerBlock) Stop() error {
	return nil
}

func (loadBalancer *LoadBalancer) AddServer(id int, server common.Server) {
	_, ok := loadBalancer.Servers[id]
	if ok {
		log.Fatalf("Server already exists")
		return
	}
	loadBalancer.Servers[id] = server
	loadBalancer.ServerList[loadBalancer.Pointer] = id
	loadBalancer.Pointer += 1
}

func (loadBalancer *LoadBalancer) GetServer() common.Server {
	server, _ := loadBalancer.Servers[loadBalancer.ServerList[loadBalancer.ServerPointer]]
	loadBalancer.ServerPointer = (loadBalancer.ServerPointer + 1) % loadBalancer.Pointer
	return server
}
