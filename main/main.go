package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
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

// -------------------------- Stage 1 usage --------------------------
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

	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.ServerAddress())
	addressChannel <- listener.Addr().String()
	testServer.AcceptConnection(listener)
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

func clientCallRPC(client *client.Client, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err := client.Call("Arithmetic.Addition", input, output, timeOutContext); err != nil {
		log.Fatal("Client RPC call Arithmetic.Addition error: ", err)
	}
	log.Printf("%d + %d = %d", input.A, input.B, output.C)
}

// -------------------------- Stage 2 usage --------------------------
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
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.ServerAddress())
	addressChannel <- listener.Addr().String()
	_ = http.Serve(listener, nil)
	//server.AcceptConnection(listener)
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

// -------------------------- Stage 3A usage --------------------------
func createLoadBalancedClientAndCall(addressA string, addressB string) {
	clientLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{"tcp@" + addressA, "tcp@" + addressB})
	loadBalancedClient := client.CreateLoadBalancedClient(clientLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	log.Printf("clientLoadBalancer: %+v", clientLoadBalancer)
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

func createLoadBalancedClientAndBroadcastCall(addressA string, addressB string) {
	clientLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{"tcp@" + addressA, "tcp@" + addressB})
	loadBalancedClient := client.CreateLoadBalancedClient(clientLoadBalancer, loadBalancer.RandomSelectMode, nil)
	defer func() { _ = loadBalancedClient.Close() }()

	var waitGroup sync.WaitGroup
	n := 0
	for n < 6 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			loadBalancedClientBroadcastCallRPC(loadBalancedClient, n)
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

func loadBalancedClientBroadcastCallRPC(loadBalancedClient *client.LoadBalancedClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect 2-5 timeout
	//noTimeOutContext := context.Background()
	//err := loadBalancedClient.BroadcastCall(noTimeOutContext, "Arithmetic.SleepThenAddition", input, output)
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*2)
	err := loadBalancedClient.BroadcastCall(timeOutContext, "Arithmetic.SleepThenAddition", input, output)
	if err != nil {
		log.Println("RPC call Arithmetic.SleepThenAddition error: ", err)
	} else {
		log.Printf("RPC call Arithmetic.SleepThenAddition success: %d + %d = %d", input.A, input.B, output.C)
	}
	//cancelContext()
}

// -------------------------- Stage 3B usage --------------------------
func createRegistry(waitGroup *sync.WaitGroup) {
	log.Println("main -> createRegistry: RPC registry initialization routine start...")
	listener, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	waitGroup.Done()
	_ = http.Serve(listener, nil)
	log.Println("main -> createRegistry: RPC registry initialization routine end...")
}

func createServerOnRegistry(registryAddress string, waitGroup *sync.WaitGroup) {
	log.Println("main -> createServerOnRegistry: RPC server initialization routine start...")
	var arithmetic Arithmetic
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	testServer := server.CreateServer(listener.Addr())
	err = testServer.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}
	testServer.Heartbeat(registryAddress, 0)
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.ServerAddress())
	waitGroup.Done()
	testServer.AcceptConnection(listener)
	log.Println("main -> createServerOnRegistry: RPC server initialization routine end...")

}

func createLoadBalancedClientAndCallOnRegistry(registryAddress string) {
	registryLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(registryAddress, 0)
	loadBalancedClient := client.CreateLoadBalancedClient(registryLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	log.Printf("registryLoadBalancer: %+v", registryLoadBalancer)
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

func createLoadBalancedClientAndBroadcastCallOnRegistry(registryAddress string) {
	registryLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(registryAddress, 0)
	loadBalancedClient := client.CreateLoadBalancedClient(registryLoadBalancer, loadBalancer.RandomSelectMode, nil)
	defer func() { _ = loadBalancedClient.Close() }()

	var waitGroup sync.WaitGroup
	n := 0
	for n < 6 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			loadBalancedClientBroadcastCallRPC(loadBalancedClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//Stage 1: TCP (normal), server address 6666, return 0
	/*	addressChannel := make(chan string)
		go createServer(addressChannel, ":6666")
		address := <-addressChannel
		createClientAndCall(address)*/

	//Stage 2: HTTP, server address 7777, not returning (so that debug page is hosted)
	/*	addressChannel := make(chan string)
		go createClientAndCallHTTP(addressChannel)
		createServerHTTP(addressChannel, ":7777")*/

	//Stage 3A: TCP, load balancing(client side), server address random (:0), return 0
	/*	addressChannelA := make(chan string)
		addressChannelB := make(chan string)
		go createServer(addressChannelA, ":0")
		go createServer(addressChannelB, ":0")
		addressA := <-addressChannelA
		addressB := <-addressChannelB
		time.Sleep(time.Second)
		createLoadBalancedClientAndCall(addressA, addressB)
		createLoadBalancedClientAndBroadcastCall(addressA, addressB)*/

	//Stage 3B: TCP, load balancing(registry side), server address random (:0), return 0
	registryAddress := "http://localhost:9999/_srpc_/registry"
	var waitGroup sync.WaitGroup

	waitGroup.Add(1)
	go createRegistry(&waitGroup)
	waitGroup.Wait()

	time.Sleep(time.Second)

	waitGroup.Add(1)
	go createServerOnRegistry(registryAddress, &waitGroup)

	//waitGroup.Add(1)
	//go createServerOnRegistry(registryAddress, &waitGroup)
	waitGroup.Wait()

	time.Sleep(time.Second)

	createLoadBalancedClientAndCallOnRegistry(registryAddress)
	//createLoadBalancedClientAndBroadcastCallOnRegistry(registryAddress)
}
