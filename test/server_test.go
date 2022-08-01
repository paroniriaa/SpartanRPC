package test

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"testing"
	"time"
)

func startServer(address chan string) {
	portNumber, err := net.Listen("tcp", "localhost:8002")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("startServer RPC server on port", portNumber.Addr())
	address <- portNumber.Addr().String()
	server.AcceptConnection(portNumber)
}

func TestServer(test *testing.T) {
	test.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	address := make(chan string)
	go startServer(address)

	connection, _ := net.Dial("tcp", <-address)
	defer func() { _ = connection.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(connection).Encode(server.DefaultConnectionInfo)
	communication := coder.NewJsonCoder(connection)

	n := 0
	for n < 5 {
		header := &coder.MessageHeader{
			ServiceDotMethod: "Test.Echo",
			SequenceNumber:   uint64(n),
		}
		request := "RPC Sequence Number " + strconv.Itoa(n)
		log.Println("Request:", request)
		_ = communication.EncodeMessageHeaderAndBody(header, request)
		_ = communication.DecodeMessageHeader(header)
		var response string
		_ = communication.DecodeMessageBody(&response)
		log.Println("Response:", response)
		n++
	}
}
