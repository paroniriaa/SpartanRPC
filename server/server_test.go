package server


import (
	"Distributed-RPC-Framework/coder"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"testing"
	"time"
)

func start(address chan string) {
	portNumber, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("start RPC server on port", portNumber.Addr())
	address <- portNumber.Addr().String()
	Connection_handle(portNumber)
}

func TestServer(test *testing.T) {
	test.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	address := make(chan string)
	go start(address)

	connection, _ := net.Dial("tcp", <-address)
	defer func() { _ = connection.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(connection).Encode(DefaultOption)
	communication := coder.NewJsonCoder(connection)

	n := 0
	for n < 5 {
		header := &coder.Header{
			ServiceMethod:  "Test.Echo",
			SequenceNumber: uint64(n),
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

