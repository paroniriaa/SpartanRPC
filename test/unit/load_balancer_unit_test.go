package unit

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"log"
	"sync"
	"testing"
	"time"
)

func TestLoadBalancer(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	var waitGroup sync.WaitGroup

	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	//create 2 servers to simulate client-side load balancing
	serverChannelA := make(chan *server.Server)
	serverChannelB := make(chan *server.Server)
	time.Sleep(time.Second)
	waitGroup.Add(2)
	go createServer(":0", serviceList, serverChannelA, &waitGroup)
	testServerA := <-serverChannelA
	log.Printf("Load balancer unit -> main: Server A address fetched from serverChannelA: %s", testServerA.ServerAddress)
	go createServer(":0", serviceList, serverChannelB, &waitGroup)
	testServerB := <-serverChannelB
	log.Printf("Load balancer unit -> main: Server B address fetched from serverChannelA: %s", testServerB.ServerAddress)
	waitGroup.Wait()

	//create client-side load balancer for testing
	clientSideLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{testServerA.ServerAddress, testServerB.ServerAddress})
	//create load balanced client for testing
	clientSideLoadBalancedClient := client.CreateLoadBalancedClient(clientSideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	defer func() { _ = clientSideLoadBalancedClient.Close() }()

	// create registry for registry-side load balancer
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":0", registryChannel, &waitGroup)
	testRegistry := <-registryChannel
	log.Printf("Load balancer unit -> main: Registry address fetched: %s", testRegistry.RegistryURL)
	waitGroup.Wait()

	//servers send heartbeat to the registry to register as alive server to simulate registry-side load balancing
	testServerA.Heartbeat(testRegistry.RegistryURL, 0)
	testServerB.Heartbeat(testRegistry.RegistryURL, 0)

	//create registry-side load balancer for testing
	registrySideLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(testRegistry.RegistryURL, 0)
	//create load balanced client for testing
	registrySideLoadBalancedClient := client.CreateLoadBalancedClient(registrySideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	defer func() { _ = registrySideLoadBalancedClient.Close() }()

	//table-driven tests
	arithmeticCases := []ArithmeticCase{
		{"Arithmetic.Addition", "+", &Input{8, 2}, &Output{}, 10},
		{"Arithmetic.Subtraction", "-", &Input{7, 3}, &Output{}, 4},
		{"Arithmetic.Multiplication", "*", &Input{6, 4}, &Output{}, 24},
		{"Arithmetic.Division", "/", &Input{5, 5}, &Output{}, 1},
	}

	//client-side load balanced asynchronous call unit
	prefix := "ClientSideBalanced.Asynchronous.NormalCalls."
	//time.Sleep(time.Second)
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createNormalCallTestCase(t, clientSideLoadBalancedClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

	//client-side load balanced asynchronous broadcast call unit
	prefix = "ClientSideBalanced.Asynchronous.BroadcastCalls."
	//time.Sleep(time.Second)
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createBroadcastCallTestCase(t, clientSideLoadBalancedClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

	//registry-side load balanced asynchronous call unit
	prefix = "RegistrySideBalanced.Asynchronous.NormalCalls."
	//time.Sleep(time.Second)
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createNormalCallTestCase(t, registrySideLoadBalancedClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

	//registry-side load balanced asynchronous broadcast call unit
	prefix = "RegistrySideBalanced.Asynchronous.BroadcastCalls."
	//time.Sleep(time.Second)
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createBroadcastCallTestCase(t, registrySideLoadBalancedClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

}
