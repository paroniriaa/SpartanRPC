package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Input struct {
	A, B int
}

type Output struct {
	C int
}

type Arithmetic int

func (t *Arithmetic) Addition(input *Input, output *Output) error {
	output.C = input.A + input.B
	return nil
}

func createServer(addressChannel chan string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("Server register error:", err)
	}
	listener, err := net.Listen("tcp", ":6666")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func createServerHTTP(addressChannel chan string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("Server register error:", err)
	}
	listener, err := net.Listen("tcp", ":7777")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	server.RegisterHandlerHTTP()
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	_ = http.Serve(listener, nil)
	//server.AcceptConnection(listener)
}

func createClientAndCall(addressChannel chan string) {
	testClient, _ := client.MakeDial("tcp", <-addressChannel)
	defer func() { _ = testClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 2 {
		waitGroup.Add(1)
		go func(n int) {
			clientCallRPC(testClient, n, &waitGroup)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func createClientAndCallHTTP(addressChannel chan string) {
	testClient, _ := client.MakeDialHTTP("tcp", <-addressChannel)
	defer func() { _ = testClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 2 {
		waitGroup.Add(1)
		//
		go func(n int) {
			clientCallRPC(testClient, n, &waitGroup)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func clientCallRPC(client *client.Client, number int, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	input := &Input{A: number, B: number ^ 2}
	output := &Output{}
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err := client.Call("Arithmetic.Addition", input, output, timeOutContext); err != nil {
		log.Fatal("Client RPC call Demo.Sum error:", err)
	}
	log.Printf("%d + %d = %d", input.A, input.B, output.C)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	addressChannel := make(chan string)

	//TCP (normal), server address 6666
	//go createServer(addressChannel)
	//createClientAndCall(addressChannel)

	//HTTP, server address 7777
	go createClientAndCallHTTP(addressChannel)
	createServerHTTP(addressChannel)
}
