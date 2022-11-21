package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"log"
	"net"
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
func createServer(port string, addressChannel chan string, waitGroup *sync.WaitGroup) {
	var arithmetic Arithmetic
	log.Println("main -> createServer: RPC server initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServer(listener)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server creation issue:", err)
	}
	err = testServer.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server register error:", err)
	}
	addressChannel <- listener.Addr().String()
	waitGroup.Done()
	log.Println("main -> createServer: RPC server initialization routine end, now launched and accepting...")
	//BLOCKING and keep listening
	testServer.LaunchAndAccept()
}

func createClientAndCall(serverAddress string) {
	testClient, _ := client.MakeDial("tcp", serverAddress)
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
func createServerHTTP(port string, addressChannel chan string, waitGroup *sync.WaitGroup) {
	var arithmetic Arithmetic
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: Server network issue: ", err)
	}
	testHTTPServer, err := server.CreateServer(listener)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: Server creation issue: ", err)
	}
	err = testHTTPServer.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: Server register error: ", err)
	}
	addressChannel <- listener.Addr().String()
	waitGroup.Done()
	//BLOCKING and keep serving
	testHTTPServer.LaunchAndServe()
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

func createServerOnRegistry(port string, registryURL string, serverChannel chan *server.Server, waitGroup *sync.WaitGroup) {
	log.Println("main -> createServerOnRegistry: RPC server initialization routine start...")
	var arithmetic Arithmetic
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServerOnRegistry error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServer(listener)
	if err != nil {
		log.Fatal("main -> createServerOnRegistry error: RPC server creation issue:", err)
	}
	err = testServer.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("main -> createServerOnRegistry error: RPC server register error:", err)
	}
	testServer.Heartbeat(registryURL, 0)
	serverChannel <- testServer
	waitGroup.Done()
	log.Println("main -> createServerOnRegistry: RPC server initialization routine end, now launched and accepting...")
	//BLOCKING and keep listening
	testServer.LaunchAndAccept()
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
	var waitGroup sync.WaitGroup

	//Stage 1:
	//Server: 1 TCP-based RPC server
	//Client: 1 TCP-based RPC client, connected to RPC server via direct Make-Dial()
	//Load Balancer: N/A
	//Registry: N/A
	//Call: 6 RPC normal calls, called from RPC client via direct Call()
	//Expected result: All RPC calls are handled, return 0

	/*	addressChannel := make(chan string)
		waitGroup.Add(1)
		go createServer(":0", addressChannel, &waitGroup)
		serverAddress := <-addressChannel
		log.Printf("main -> main: Server address fetched from addressChannel: %s", serverAddress)
		waitGroup.Wait()
		createClientAndCall(serverAddress)*/

	//Stage 2:
	//Server: 1 HTTP-based RPC server
	//Client: 1 HTTP-based RPC client, connected to RPC server via direct Make-Dial-HTTP()
	//Load Balancer: N/A
	//Registry: N/A
	//Call: 6 RPC normal calls, called from RPC client via direct Call()
	//Expected result: All RPC calls are handled, not returning (so that debug page is hosted)

	/*	addressChannel := make(chan string)
		go createClientAndCallHTTP(addressChannel)
		waitGroup.Add(1)
		createServerHTTP(":0", addressChannel, &waitGroup)
		waitGroup.Wait()*/

	//Stage 3A:
	//Server: 2 TCP-based RPC server
	//Client: 2 TCP-based RPC client, controlled by 1 RPC client manager
	//Load Balancer: 1 TCP-based client-side load balancer, manual server update, no registry discover
	//Registry: N/A
	//Call: 6 RPC normal calls, 6 RPC broadcast calls (12 normal calls for 2 server), called from RPC client manager via indirect Call()
	//Expected result: All 6 RPC normal calls are handled, 2 RPC broadcast calls are handled, 4 RPC broadcast calls are timeout, return 0

	/*	addressChannelA := make(chan string)
		addressChannelB := make(chan string)
		time.Sleep(time.Second)
		waitGroup.Add(2)
		go createServer(":0", addressChannelA, &waitGroup)
		addressA := <-addressChannelA
		log.Printf("main -> main: Server A address fetched from addressChannel: %s", addressA)
		go createServer(":0", addressChannelB, &waitGroup)
		addressB := <-addressChannelB
		waitGroup.Wait()
		log.Printf("main -> main: Server B address fetched from addressChannel: %s", addressB)
		createLoadBalancedClientAndCall(addressA, addressB)
		createLoadBalancedClientAndBroadcastCall(addressA, addressB)*/

	//Stage 3B:
	//Server: 2 TCP-based RPC server, auto heartbeat sending
	//Client: 2 TCP-based RPC client, controlled by 1 RPC client manager
	//Load Balancer: 1 TCP-based registry-side load balancer, auto server update, service discover on
	//Registry: 1 HTTP-based RPC registry, auto server refresh, heartbeat monitoring on, service dispatch on
	//Call: 6 RPC normal calls, 6 RPC broadcast calls (12 normal calls for 2 server), called from RPC client manager via indirect Call()
	//Expected result: All 6 RPC normal calls are handled, 2 RPC broadcast calls are handled, 4 RPC broadcast calls are timeout, return 0

	registryChannel := make(chan *registry.Registry)
	time.Sleep(time.Second)
	waitGroup.Add(1)
	go createRegistry(":0", registryChannel, &waitGroup)
	testRegistry := <-registryChannel
	log.Printf("main -> main: Registry address fetched from registryChannel: %s", testRegistry.RegistryURL)
	waitGroup.Wait()
	serverChannelC := make(chan *server.Server)
	serverChannelD := make(chan *server.Server)
	waitGroup.Add(2)
	go createServerOnRegistry(":0", testRegistry.RegistryURL, serverChannelC, &waitGroup)
	serverC := <-serverChannelC
	log.Printf("main -> main: Server C address fetched from serverChannelC: %s", serverC.ServerAddress)
	go createServerOnRegistry(":0", testRegistry.RegistryURL, serverChannelD, &waitGroup)
	serverD := <-serverChannelD
	log.Printf("main -> main: Server D address fetched from serverChannelD: %s", serverD.ServerAddress)
	waitGroup.Wait()
	createLoadBalancedClientAndCallOnRegistry(testRegistry.RegistryURL)
	createLoadBalancedClientAndBroadcastCallOnRegistry(testRegistry.RegistryURL)
}
