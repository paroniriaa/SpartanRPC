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

func (t *Arithmetic) SleepThenAddition(input *Input, output *Output) error {
	time.Sleep(time.Second * time.Duration(input.A))
	output.C = input.A + input.B
	return nil
}

func createServer(addressChannel chan string, addressPort string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func createServerHTTP(addressChannel chan string, addressPort string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}
	listener, err := net.Listen("tcp", addressPort)
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
			defer waitGroup.Done()
			clientCallRPC(testClient, n)
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
		go func(n int) {
			defer waitGroup.Done()
			clientCallRPC(testClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func clientCallRPC(client *client.Client, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err := client.Call("Arithmetic.Addition", input, output, timeOutContext); err != nil {
		log.Fatal("Client RPC call Arithmetic.Addition error: ", err)
	}
	log.Printf("%d + %d = %d", input.A, input.B, output.C)
}

func createDiscoveryClientAndCall(addressChannelA chan string, addressChannelB chan string) {
	addressA := <-addressChannelA
	addressB := <-addressChannelB
	discovery := client.CreateDiscoveryMultiServer([]string{"tcp@" + addressA, "tcp@" + addressB})
	testDiscoveryClient := client.CreateDiscoveryClient(discovery, client.RandomSelectMode, nil)
	defer func() { _ = testDiscoveryClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 5 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			discoveryClientCallRPC(testDiscoveryClient, n)
			discoveryClientBroadcastRPC(testDiscoveryClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func createDiscoveryClientAndBroadcast(addressChannelA chan string, addressChannelB chan string) {
	addressA := <-addressChannelA
	addressB := <-addressChannelB
	discovery := client.CreateDiscoveryMultiServer([]string{"tcp@" + addressA, "tcp@" + addressB})
	testDiscoveryClient := client.CreateDiscoveryClient(discovery, client.RandomSelectMode, nil)
	defer func() { _ = testDiscoveryClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 5 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			discoveryClientBroadcastRPC(testDiscoveryClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func discoveryClientCallRPC(discoveryClient *client.DiscoveryClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect no timeout
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*100)
	if err := discoveryClient.Call(timeOutContext, "Arithmetic.Addition", input, output); err != nil {
		log.Fatal("discoveryClient RPC call Arithmetic.Addition error: ", err)
	}
	log.Printf("%s %s success: %d + %d = %d", "call", "Arithmetic.Addition", input.A, input.B, output.C)
}

func discoveryClientBroadcastRPC(discoveryClient *client.DiscoveryClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect 2-5 timeout
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*2)
	if err := discoveryClient.Broadcast(timeOutContext, "Arithmetic.SleepThenAddition", input, output); err != nil {
		log.Fatal("discoveryClient RPC call Arithmetic.SleepThenAddition error: ", err)
	}
	log.Printf("%s %s success: %d + %d = %d", "call", "Arithmetic.SleepThenAddition", input.A, input.B, output.C)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//TCP (normal), server address 6666
	//addressChannel := make(chan string)
	//go createServer(addressChannel, ":6666")
	//createClientAndCall(addressChannel)

	//HTTP, server address 7777
	//addressChannel := make(chan string)
	//go createClientAndCallHTTP(addressChannel)
	//createServerHTTP(addressChannel, ":7777")

	addressChannelA := make(chan string)
	addressChannelB := make(chan string)
	go createServer(addressChannelA, ":0")
	go createServer(addressChannelB, ":0")
	time.Sleep(time.Second)
	createDiscoveryClientAndCall(addressChannelA, addressChannelB)
	createDiscoveryClientAndBroadcast(addressChannelA, addressChannelB)

}
