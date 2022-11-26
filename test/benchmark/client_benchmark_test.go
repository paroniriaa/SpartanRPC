package benchmark

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

func BenchmarkClient(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	time.Sleep(time.Second)

	//create test needed variables
	var waitGroup sync.WaitGroup
	var testRegistry *registry.Registry
	var testServerA *server.Server
	var testServerB *server.Server
	var testServerC *server.Server
	var testCase ArithmeticCase
	var timeOutContext context.Context
	var testClient *client.Client
	var testClientHTTP *client.Client
	var connectionInfo *server.ConnectionInfo
	var connectionInfoHTTP *server.ConnectionInfo
	var connection net.Conn
	var connectionHTTP net.Conn

	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	//create registry for client usage testing
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":8001", registryChannel, &waitGroup)
	testRegistry = <-registryChannel
	waitGroup.Wait()

	//create 3 servers A and B, and C(HTTP) for testing
	serverChannelA := make(chan *server.Server)
	serverChannelB := make(chan *server.Server)
	serverChannelC := make(chan *server.Server)
	waitGroup.Add(3)
	go createServer(":9001", serviceList, serverChannelA, &waitGroup)
	testServerA = <-serverChannelA
	go createServer(":9002", serviceList, serverChannelB, &waitGroup)
	testServerB = <-serverChannelB
	go createServerHTTP(":9003", serviceList, serverChannelC, &waitGroup)
	testServerC = <-serverChannelC
	waitGroup.Wait()

	testServerA.Heartbeat(testRegistry.RegistryURL, 0)
	testServerB.Heartbeat(testRegistry.RegistryURL, 0)
	//testServerC.Heartbeat(testRegistry.RegistryURL, 0)

	//test case for client to send to server
	testCase = ArithmeticCase{
		"Arithmetic.Addition",
		"+",
		&Input{1, 1},
		&Output{},
		2,
	}
	//timeout context for test case
	timeOutContext = context.Background()

	//testClient for testing Client.Call()
	testClient, _ = client.XMakeDial(testServerA.ServerAddress)

	//testClientHTTP for testing ClientHTTP.Call()
	testClientHTTP, _ = client.XMakeDial(testServerC.ServerAddress)

	//client-side load balancer for testing client-side load balanced client Call(), BroadcastCall()
	clientSideLoadBalancer := loadBalancer.CreateLoadBalancerClientSide([]string{testServerA.ServerAddress, testServerB.ServerAddress})
	clientSideLoadBalancedClient := client.CreateLoadBalancedClient(clientSideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)

	//registry-side load balancer for testing registry-side load balanced client Call(), BroadcastCall()
	registrySideLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(testRegistry.RegistryURL, 0)
	registrySideLoadBalancedClient := client.CreateLoadBalancedClient(registrySideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)

	//create connection and connectionInfo for simulating client dial to server
	connectionInfo, _ = client.ParseConnectionInfo()
	connection, _ = net.DialTimeout("tcp", testServerA.Listener.Addr().String(), connectionInfo.ConnectionTimeout)

	//create connection and connectionInfo for simulating client dial to server (HTTP)
	connectionInfoHTTP, _ = client.ParseConnectionInfo()
	connectionHTTP, _ = net.DialTimeout("tcp", testServerC.Listener.Addr().String(), connectionInfo.ConnectionTimeout)

	b.Run("CreateClient", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.CreateClient(connection, connectionInfo)
		}
	})

	b.Run("CreateClient.HTTP", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.CreateClientHTTP(connectionHTTP, connectionInfoHTTP)
		}
	})

	b.Run("Call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testClient.Call(testCase.ServiceDotMethod, testCase.Input, testCase.Output, timeOutContext)
		}
	})

	b.Run("Call.HTTP", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testClientHTTP.Call(testCase.ServiceDotMethod, testCase.Input, testCase.Output, timeOutContext)
		}
	})

	b.Run("XMakeDial", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.XMakeDial(testServerA.ServerAddress)
		}
	})

	b.Run("XMakeDial.HTTP", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.XMakeDial(testServerC.ServerAddress)
		}
	})

	b.Run("MakeDial", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.MakeDial("tcp", testServerA.Listener.Addr().String())
		}
	})

	b.Run("MakeDial.HTTP", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.MakeDialHTTP("tcp", testServerC.Listener.Addr().String())
		}
	})

	b.Run("MakeDialWithTimeout", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.MakeDialWithTimeout(client.CreateClient, "tcp", testServerA.Listener.Addr().String())
		}
	})

	b.Run("MakeDialWithTimeout.HTTP", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.MakeDialWithTimeout(client.CreateClientHTTP, "tcp", testServerC.Listener.Addr().String())
		}
	})

	b.Run("Client-Side.CreateLoadBalancedClient", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.CreateLoadBalancedClient(clientSideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
		}
	})

	b.Run("Client-Side.LoadBalancedClient.Call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clientSideLoadBalancedClient.Call(timeOutContext, testCase.ServiceDotMethod, testCase.Input, testCase.Output)
		}
	})

	b.Run("Client-Side.LoadBalancedClient.BroadcastCall", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clientSideLoadBalancedClient.BroadcastCall(timeOutContext, testCase.ServiceDotMethod, testCase.Input, testCase.Output)
		}
	})

	b.Run("Registry-Side.CreateLoadBalancedClient", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.CreateLoadBalancedClient(registrySideLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
		}
	})

	b.Run("Registry-Side.LoadBalancedClient.Call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registrySideLoadBalancedClient.Call(timeOutContext, testCase.ServiceDotMethod, testCase.Input, testCase.Output)
		}
	})

	b.Run("Registry-Side.LoadBalancedClient.BroadcastCall", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registrySideLoadBalancedClient.BroadcastCall(timeOutContext, testCase.ServiceDotMethod, testCase.Input, testCase.Output)
		}
	})

}
