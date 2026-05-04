// first point of contact for any api. Performs TLS handshakes and gives it forward to load balancer

package api_gateway

import (
	"fmt"
	"net"
	"crypto/tls"
	"log"
	"bufio"
	"io"
	"strings"
	"strconv"
	"time"
	"encoding/json"
	"github.com/thatrajaryan/web-server/common"
	"github.com/thatrajaryan/web-server/load_balancer"
)

func api_gateway() {
	// 1. Load your certificate and private key
	cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
    if err != nil {
        log.Fatalf("Failed to load certificates: %v", err)
    }
	// 2. Define the TLS configuration
    config := &tls.Config{Certificates: []tls.Certificate{cert}}

    // 3. Create a TLS listener instead of net.Listen
    ln, err := tls.Listen("tcp", ":8080", config)
    if err != nil {
        log.Fatalf("Failed to start listener: %v", err)
    }
    defer ln.Close()

	session_timestamp := make(map[int]int64)
	for {
		// Accept incoming connection
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("Failed to Accept Connection: %v", err)
			continue
		}

		// handling connection as a goroutine
		go handleConnection(conn, session_timestamp)
	}
}

func handleConnection(conn net.Conn, session_timestamp map[int]int64) {
	defer conn.Close()
	
	reader := bufio.NewReader(conn)

	// reader contains HTTPS message
	httpRequest := httpParser(reader)

	session_id, err := strconv.Atoi(httpRequest.Header["Session-ID"])
	if err != nil {
		log.Fatalf("Failed to Accept Connection: %v", err)
		return
	}
	value, ok := session_timestamp[session_id]

	if !ok {
		// timeout is 1 hr
		session_timestamp[session_id] = time.Now().UnixMilli() + int64(60*60*1000)
	} else if time.Now().UnixMilli() > value {
		response := common.HttpResponse{ Status: 408, Message: "Connection timed out\n" }
		data, err := json.Marshal(response)
		if err != nil {
			log.Fatalf("Failed to Accept Connection: %v", err)
			return
		}
		conn.Write(data)
		return
	}

	loadBalancer := load_balancer.LoadBalancer{ 
		Servers: map[int]common.Server{ 
			0: common.Server{ IpAddress: "localhost", Port: 8081},
			1: common.Server{ IpAddress: "localhost", Port: 8082},
			2: common.Server{ IpAddress: "localhost", Port: 8083},
			3: common.Server{ IpAddress: "localhost", Port: 8084},
			4: common.Server{ IpAddress: "localhost", Port: 8085},
		},
		ServerList: []int{0, 1, 2, 3, 4},
		Pointer: 5,
		ServerPointer: 0,
	}
	server := loadBalancer.GetServer()
	fmt.Printf("Connecting to Server: %s:%d", server.IpAddress, server.Port)
	response := common.HttpResponse{ Status: 201, Message: "Message Received by Server\n" }
	data, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Failed to Accept Connection: %v", err)
		return
	}
	conn.Write(data)
}

func httpParser(reader *bufio.Reader) common.HttpRequest {
	_, err := reader.ReadString('\n') // method, endpoint, version

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to Accept Connection: %v", err)
			break
		}

		// marking end of header?
		if line == "\r\n" {
			break
		}
		line = strings.TrimSuffix(line, "\r\n")
		parts := strings.SplitN(line, ": ", 2)
		headers[parts[0]] = parts[1]
	}
	bodySize, err := strconv.Atoi(headers["Content-Length"])
	if err != nil {
		log.Fatalf("Failed to Parse content length: %v", err)
		return common.HttpRequest{}
	}
	bodyBuffer := make([]byte, bodySize)
	_, err = io.ReadFull(reader, bodyBuffer)

	if err != nil {
		log.Fatalf("Failed to Accept Connection: %v", err)
		return common.HttpRequest{}
	}
	body := string(bodyBuffer)
	return common.HttpRequest{ Header: headers, Body: body }
}