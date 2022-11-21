package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"log"
	"sync"
	"testing"
)

func TestLoadBalancer(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	var waitGroup sync.WaitGroup
	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	services := []any{
		&arithmetic,
	}

	//create 2 servers to simulate client-side load balancing
	addressChannelA := make(chan string)
	addressChannelB := make(chan string)
	waitGroup.Add(2)
	go createServer(":0", services, addressChannelA, &waitGroup)
	serverAddressA := <-addressChannelA
	log.Printf("Load balancer test -> main: Server A (not registered) address fetched: %s", serverAddressA)
	go createServer(":0", services, addressChannelB, &waitGroup)
	serverAddressB := <-addressChannelB
	log.Printf("Load balancer test -> main: Server B (not registered) address fetched: %s", serverAddressB)
	waitGroup.Wait()

	//create client-side load balancer for testing
	clientSideLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{"tcp@" + serverAddressA, "tcp@" + serverAddressB})
	//create load balanced client for testing
	clientSideLoadBalancedClient := client.CreateLoadBalancedClient(clientSideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	defer func() { _ = clientSideLoadBalancedClient.Close() }()

	// create registry for registry-side load balancer
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":0", registryChannel, &waitGroup)
	testRegistry := <-registryChannel
	log.Printf("Load balancer test -> main: Registry address fetched: %s", testRegistry.RegistryURL)
	waitGroup.Wait()

	//create 2 servers to simulate registry-side load balancing
	serverChannelC := make(chan *server.Server)
	serverChannelD := make(chan *server.Server)
	waitGroup.Add(2)
	go createServerOnRegistry(":0", testRegistry.RegistryURL, services, serverChannelC, &waitGroup)
	serverC := <-serverChannelC
	log.Printf("Load balancer test -> main: Server C (registered) address fetched: %s", serverC.ServerAddress)
	go createServerOnRegistry(":0", testRegistry.RegistryURL, services, serverChannelD, &waitGroup)
	serverD := <-serverChannelD
	log.Printf("Load balancer test -> main: Server D (registered) address fetched: %s", serverD.ServerAddress)
	waitGroup.Wait()

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

	//client-side load balanced asynchronous call test
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

	//client-side load balanced asynchronous broadcast call test
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

	//registry-side load balanced asynchronous call test
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

	//registry-side load balanced asynchronous broadcast call test
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
