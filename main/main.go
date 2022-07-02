package main

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"time"
)

func start(address chan string) {
	portNumber, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("start RPC server on port", portNumber.Addr())
	address <- portNumber.Addr().String()
	server.Connection_handle(portNumber)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	address := make(chan string)
	go start(address)

	connection, _ := net.Dial("tcp", <-address)
	defer func() { _ = connection.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(connection).Encode(server.DefaultOption)
	communication := coder.NewJsonCoder(connection)

	n := 0
	for n < 5 {
		requestHeader := &coder.Header{
			ServiceMethod:  "Test.Echo",
			SequenceNumber: uint64(n),
		}
		requestBody := "Hello there! " + strconv.Itoa(n)
		log.Printf("Request: ServiceMethod -> %s, SequenceNumber -> %d, Message -> %s", requestHeader.ServiceMethod, requestHeader.SequenceNumber, requestBody)
		_ = communication.EncodeMessageHeaderAndBody(requestHeader, requestBody)

		responseHeader := &coder.Header{}
		var responseBody string
		_ = communication.DecodeMessageHeader(responseHeader)
		_ = communication.DecodeMessageBody(&responseBody)
		log.Printf("Response: ServiceMethod -> %s, SequenceNumber -> %d, Message -> %s", responseHeader.ServiceMethod, responseHeader.SequenceNumber, responseBody)
		n++
	}
}
