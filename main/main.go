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
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	testServer := server.CreateServer(listener.Addr())
	err = testServer.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}

	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	testServer.AcceptConnection(listener)
}

func createServerHTTP(addressChannel chan string, addressPort string) {
	var arithmetic Arithmetic
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	testHTTPServer := server.CreateServer(listener.Addr())
	err = testHTTPServer.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}
	testHTTPServer.RegisterHandlerHTTP()
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	_ = http.Serve(listener, nil)
	//server.AcceptConnection(listener)
}

func createClientAndCall(address string) {
	testClient, _ := client.MakeDial("tcp", address)
	defer func() { _ = testClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 6 {
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
	for n < 6 {
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

func createLoadBalancedClientAndCall(addressA string, addressB string) {
	loadBalancer := client.CreateLoadBalancerClientSide([]string{"tcp@" + addressA, "tcp@" + addressB})
	loadBalancedClient := client.CreateLoadBalancedClient(loadBalancer, client.RoundRobinSelectMode, nil)
	log.Printf("loadBalancer: %+v", loadBalancer)
	log.Printf("Before -> loadBalancedClient: %+v", loadBalancedClient)
	defer func() { _ = loadBalancedClient.Close() }()

	var waitGroup sync.WaitGroup
	n := 0
	for n < 6 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			loadBalancedClientCallRPC(loadBalancedClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
	log.Printf("After -> loadBalancedClient: %+v", loadBalancedClient)
}

func createLoadBalancedClientAndBroadcast(addressA string, addressB string) {
	loadBalancer := client.CreateLoadBalancerClientSide([]string{"tcp@" + addressA, "tcp@" + addressB})
	loadBalancedClient := client.CreateLoadBalancedClient(loadBalancer, client.RandomSelectMode, nil)
	defer func() { _ = loadBalancedClient.Close() }()

	var waitGroup sync.WaitGroup
	n := 0
	for n < 6 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			loadBalancedClientBroadcastRPC(loadBalancedClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func loadBalancedClientCallRPC(loadBalancedClient *client.LoadBalancedClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect no timeout
	err := loadBalancedClient.Call(context.Background(), "Arithmetic.Addition", input, output)
	if err != nil {
		log.Println("RPC call Arithmetic.Addition error: ", err)
	} else {
		log.Printf("RPC call Arithmetic.Addition success: %d + %d = %d", input.A, input.B, output.C)
	}
}

func loadBalancedClientBroadcastRPC(loadBalancedClient *client.LoadBalancedClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect 2-5 timeout
	//noTimeOutContext := context.Background()
	//err := loadBalancedClient.Broadcast(noTimeOutContext, "Arithmetic.SleepThenAddition", input, output)
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*2)
	err := loadBalancedClient.Broadcast(timeOutContext, "Arithmetic.SleepThenAddition", input, output)
	if err != nil {
		log.Println("RPC call Arithmetic.SleepThenAddition error: ", err)
	} else {
		log.Printf("RPC call Arithmetic.SleepThenAddition success: %d + %d = %d", input.A, input.B, output.C)
	}
	//cancelContext()
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//TCP (normal), server address 6666, return 0
	/*	addressChannel := make(chan string)
		go createServer(addressChannel, ":6666")
		address := <-addressChannel
		createClientAndCall(address)*/

	//HTTP, server address 7777, not returning (so that debug page is hosted)
	/*	addressChannel := make(chan string)
		go createClientAndCallHTTP(addressChannel)
		createServerHTTP(addressChannel, ":7777")*/

	//TCP, load balancing, server address random (:0), return 0
	addressChannelA := make(chan string)
	addressChannelB := make(chan string)
	go createServer(addressChannelA, ":0")
	go createServer(addressChannelB, ":0")
	addressA := <-addressChannelA
	addressB := <-addressChannelB
	time.Sleep(time.Second)
	createLoadBalancedClientAndCall(addressA, addressB)
	createLoadBalancedClientAndBroadcast(addressA, addressB)

}
