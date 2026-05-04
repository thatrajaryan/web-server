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
