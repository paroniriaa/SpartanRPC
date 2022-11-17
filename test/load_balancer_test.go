package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"log"
	"sync"
	"testing"
	"time"
)

func TestLoadBalancer(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	services := []any{
		&arithmetic,
	}

	//create 2 servers to simulate load balancing
	addressChannelA := make(chan string)
	addressChannelB := make(chan string)
	go createServer(addressChannelA, ":0", services)
	go createServer(addressChannelB, ":0", services)
	addressA := <-addressChannelA
	addressB := <-addressChannelB
	time.Sleep(time.Second)

	//create client-side load balancer for testing
	clientLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{"tcp@" + addressA, "tcp@" + addressB})
	//create load balanced client for testing
	loadBalancedClient := client.CreateLoadBalancedClient(clientLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	defer func() { _ = loadBalancedClient.Close() }()

	//table-driven tests
	arithmeticCases := []ArithmeticCase{
		{"Arithmetic.Addition", "+", &Input{8, 2}, &Output{}, 10},
		{"Arithmetic.Subtraction", "-", &Input{7, 3}, &Output{}, 4},
		{"Arithmetic.Multiplication", "*", &Input{6, 4}, &Output{}, 24},
		{"Arithmetic.Division", "/", &Input{5, 5}, &Output{}, 1},
	}

	var waitGroup sync.WaitGroup

	//load balancer client-side asynchronous call test
	prefix := "ClientSideBalanced.Asynchronous.NormalCalls."
	time.Sleep(time.Second)
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createNormalCallTestCase(t, loadBalancedClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

	//load balancer client-side asynchronous broadcast call test
	prefix = "ClientSideBalanced.Asynchronous.BroadcastCalls."
	time.Sleep(time.Second)
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createBroadcastCallTestCase(t, loadBalancedClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

}
