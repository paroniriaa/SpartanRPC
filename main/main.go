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
		header := &coder.Header{
			ServiceMethod:  "Test.Echo",
			SequenceNumber: uint64(n),
		}
		request := "RPC Sequence Number " + strconv.Itoa(n)
		log.Println("Request:", request)
		_ = communication.Write(header, request)
		_ = communication.ReadHeader(header)
		var response string
		_ = communication.ReadBody(&response)
		log.Println("Response:", response)
		n++
	}
}
