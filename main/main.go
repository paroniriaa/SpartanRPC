package main

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func start(address chan string) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("start RPC server on port", listener.Addr())
	address <- listener.Addr().String()
	server.Connection_handle(listener)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	//log.SetFlags(0)
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
			ServiceMethod: "Test.Echo",
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
