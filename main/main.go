package main

import (
	"Distributed-RPC-Framework"
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func start(address chan string) {
	portNumber, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("start RPC server on port", portNumber.Addr())
	address <- portNumber.Addr().String()
	server.Accept(portNumber)
}

func main() {
	log.SetFlags(0)
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
			ServiceMethod: "Foo.Sum",
			SeqNumber:     uint64(n),
		}
		n++
		_ = communication.Write(header,
			fmt.Sprintf("RPC Sequence Number #{header.SeqNumber}"))
		_ = communication.ReadHeader(header)
		var response string
		_ = communication.ReadBody(&response)
		log.Println("Response:", response)
	}
}
