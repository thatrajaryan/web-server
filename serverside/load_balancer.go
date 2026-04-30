/*
Load Balancer that receives multiple server instances to redirect to
Here we are just using Round Robin
*/

package serverside

import (
	"log"
)

type Server struct {
	Ipaddress string
	Port int
}

type LoadBalancer struct {
	Servers map[int]Server
	ServerList [1000]int
	Pointer int
	ServerPointer int
}

func (loadBalancer LoadBalancer) addServer(id int, server Server) {
	_, ok := loadBalancer.Servers[id]
	if ok {
		log.Fatalf("Server already exists")
		return
	}
	loadBalancer.Servers[id] = server
	loadBalancer.ServerList[loadBalancer.Pointer] = id
	loadBalancer.Pointer += 1
}

func (loadBalancer LoadBalancer) getServer() Server {
	server, _ := loadBalancer.Servers[loadBalancer.ServerList[loadBalancer.ServerPointer]]
	loadBalancer.ServerPointer = (loadBalancer.ServerPointer + 1) % loadBalancer.Pointer
	return server
}