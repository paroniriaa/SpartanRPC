package test

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"sync"
	"testing"
)

func TestServer(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	var waitGroup sync.WaitGroup
	//create service type (arithmetic is enough)
	var test Test
	services := []any{
		&test,
	}

	addressChannel := make(chan string)
	waitGroup.Add(1)
	go createServer(":0", services, addressChannel, &waitGroup)
	serverAddress := <-addressChannel
	log.Printf("Server test -> main: Server address fetched: %s", serverAddress)
	waitGroup.Wait()

	connection, _ := net.Dial("tcp", serverAddress)
	defer func() { _ = connection.Close() }()

	_ = json.NewEncoder(connection).Encode(server.DefaultConnectionInfo)
	communication := coder.NewJsonCoder(connection)

	n := 0
	for n < 5 {
		requestHeader := &coder.MessageHeader{
			ServiceDotMethod: "Test.Echo",
			SequenceNumber:   uint64(n),
		}
		requestBody := "RPC Sequence Number " + strconv.Itoa(n)
		log.Println("Request:", requestBody)
		_ = communication.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		responseHeader := &coder.MessageHeader{}
		_ = communication.DecodeMessageHeader(responseHeader)
		var responseBody string
		_ = communication.DecodeMessageBody(&responseBody)
		log.Println("Response:", responseBody)
		n++
	}
}
