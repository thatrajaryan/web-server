package clientside

import (
    "fmt"
	"crypto/tls"
    "encoding/json"
    "github.com/thatrajaryan/webserver/common"
)

func main() {
	// 1. Define the TLS configuration
    config := &tls.Config{
        // For local development with self-signed certs, you might need this:
        InsecureSkipVerify: true, 
        // Or set the expected hostname if not using 'localhost'
        ServerName: "localhost", 
    }
    // Connect to the server
    conn, err := tls.Dial("tcp", "localhost:8080", config)
    if err != nil {
        fmt.Println(err)
        return
    }

	httpRequest := createHttp()
    // Send some data to the server
    _, err = conn.Write([]byte(httpRequest))
    if err != nil {
        fmt.Println(err)
        return
    }

	buf := make([]byte, 1024)
	length, err := conn.Read(buf)

	if err != nil {
        fmt.Println(err)
        return
    }

    var httpResponse common.HttpResponse
    err = json.Unmarshal(buf[:length], &httpResponse)
    if err != nil {
        fmt.Println(err)
        return
    }

	fmt.Printf("[Client] Message Received : %s", httpResponse.Message)

    // Close the connection
    conn.Close()
}

func createHttp() string {
	request := "GET /index.html HTTP/1.1\r\n" +
           "Host: example.com\r\n" +
		   "Content-Type: application/json\r\n" +
           "Content-Length: 15\r\n" +
           "Session-ID: 12345\r\n" + 
           "\r\n" +
		   `{"id": "12345"}`
	
	return request
}