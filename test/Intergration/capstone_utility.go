package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

type ArithmeticCase struct {
	ServiceDotMethod string
	ArithmeticSymbol string
	Input            *Input
	Output           *Output
	Expected         int
}

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

func (t *Arithmetic) Subtraction(input *Input, output *Output) error {
	output.C = input.A - input.B
	return nil
}

func (t *Arithmetic) Multiplication(input *Input, output *Output) error {
	output.C = input.A * input.B
	return nil
}

func (t *Arithmetic) Division(input *Input, output *Output) error {
	if input.B == 0 {
		return errors.New("divide by zero")
	}
	output.C = input.A / input.B
	return nil
}

func (t *Arithmetic) SleepThenAddition(input *Input, output *Output) error {
	time.Sleep(time.Second * time.Duration(input.A))
	output.C = input.A + input.B
	return nil
}

// -------------------------- Stage 1 usage --------------------------
func createServer(port string, serviceList []any, serverChannel chan *server.Server, waitGroup *sync.WaitGroup) {
	log.Println("main -> createServer: RPC server initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServer(listener)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server creation issue:", err)
	}
	for _, service := range serviceList {
		err = testServer.ServiceRegister(service)
		if err != nil {
			log.Fatal("main -> createServer error: RPC server register error:", err)
		}
	}
	serverChannel <- testServer
	waitGroup.Done()
	log.Println("main -> createServer: RPC server initialization routine end, now launched and accepting...")
	//BLOCKING and keep listening
	testServer.LaunchAndAccept()
}

func createClientAndCall(serverAddress string) {
	testClient, _ := client.XMakeDial(serverAddress)
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
		log.Fatal("main -> clientCallRPC: Client RPC call Arithmetic.Addition error: ", err)
	}
	log.Printf("%d + %d = %d", input.A, input.B, output.C)
}

// -------------------------- Stage 2 usage --------------------------

func createServerHTTP(port string, serviceList []any, serverChannelHTTP chan *server.Server, waitGroup *sync.WaitGroup) {
	log.Println("main -> createServerHTTP: RPC HTTP server initialization routine start...")
	//even though we are creating HTTP-based RPC server, we still need to declare the underlying transportation protocol(TCP) when listening
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: RPC HTTP server network issue: ", err)
	}
	testServerHTTP, err := server.CreateServerHTTP(listener)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: RPC HTTP server creation issue: ", err)
	}
	for _, service := range serviceList {
		err = testServerHTTP.ServiceRegister(service)
		if err != nil {
			log.Fatal("main -> createServerHTTP error: RPC HTTP server register error:", err)
		}
	}
	serverChannelHTTP <- testServerHTTP
	waitGroup.Done()
	log.Println("main -> createServerHTTP: RPC HTTP server initialization routine end, now launched and accepting...")
	//BLOCKING and keep serving
	testServerHTTP.LaunchAndServe()
}

func createClientAndCallHTTP(serverChannelHTTP chan *server.Server) {
	testServerHTTP := <-serverChannelHTTP
	log.Printf("main -> createClientAndCallHTTP: RPC HTTP Server address fetched from serverChannelHTTP: %s", testServerHTTP.ServerAddress)
	testClient, _ := client.XMakeDial(testServerHTTP.ServerAddress)
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
func createLoadBalancedClientAndCall(serverAddressA string, serverAddressB string) {
	clientLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{serverAddressA, serverAddressB})
	loadBalancedClient := client.CreateLoadBalancedClient(clientLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	//log.Printf("Before Call -> clientLoadBalancer: %+v", clientLoadBalancer)
	//log.Printf("Before Call -> loadBalancedClient: %+v", loadBalancedClient)
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
	//log.Printf("After Call -> clientLoadBalancer: %+v", clientLoadBalancer)
	//log.Printf("After Call -> loadBalancedClient: %+v", loadBalancedClient)
}

func createLoadBalancedClientAndBroadcastCall(serverAddressA string, serverAddressB string) {
	clientLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{serverAddressA, serverAddressB})
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
		log.Println("main -> main: RPC call Arithmetic.Addition error: ", err)
	} else {
		log.Printf("main -> main: RPC call Arithmetic.Addition success: %d + %d = %d", input.A, input.B, output.C)
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
		log.Println("main -> main: RPC call Arithmetic.SleepThenAddition error: ", err)
	} else {
		log.Printf("main -> main: RPC call Arithmetic.SleepThenAddition success: %d + %d = %d", input.A, input.B, output.C)
	}
	//cancelContext()
}

// -------------------------- Stage 3B usage --------------------------
func createRegistry(port string, registryChannel chan *registry.Registry, waitGroup *sync.WaitGroup) {
	log.Println("main -> createRegistry: RPC registry initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createRegistry error: RPC registry network issue:", err)
	}
	testRegistry, err := registry.CreateRegistry(listener, registry.DefaultTimeout)
	if err != nil {
		log.Fatal("main -> createServer error: RPC registry creation issue:", err)
	}
	registryChannel <- testRegistry
	waitGroup.Done()
	log.Println("main -> createRegistry: RPC registry initialization routine end, now launched and serving...")
	//BLOCKING and keep serving
	testRegistry.LaunchAndServe()
}

func createLoadBalancedClientAndCallOnRegistry(registryURL string) {
	registryLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(registryURL, 0)
	loadBalancedClient := client.CreateLoadBalancedClient(registryLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	//serverList, _ := registryLoadBalancer.GetServerList()
	//log.Printf("Before Call -> registryLoadBalancer.serverList: %+v", serverList)
	//log.Printf("Before Call -> loadBalancedClient: %+v", loadBalancedClient)
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
	//serverList, _ = registryLoadBalancer.GetServerList()
	//log.Printf("After Call -> registryLoadBalancer.serverList: %+v", serverList)
	//log.Printf("After Call -> loadBalancedClient: %+v", loadBalancedClient)
}

func createLoadBalancedClientAndBroadcastCallOnRegistry(registryURL string) {
	registryLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(registryURL, 0)
	shortProcessTimeoutConnectionInfo := &server.ConnectionInfo{ConnectionTimeout: time.Second * 2, ProcessingTimeout: time.Second * 2}
	loadBalancedClient := client.CreateLoadBalancedClient(registryLoadBalancer, loadBalancer.RoundRobinSelectMode, shortProcessTimeoutConnectionInfo)
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
