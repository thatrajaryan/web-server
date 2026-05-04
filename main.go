package main

import (
    "fmt"
	"github.com/thatrajaryan/web-server/api_gateway"
    "github.com/thatrajaryan/web-server/server"
)

func main() {
    fmt.Println("Initalizing Servers")
    go server.Initialize()
    fmt.Println("Forming API Gateway and Load Balancer Services")
	go api_gateway.ApiGateway()
}
