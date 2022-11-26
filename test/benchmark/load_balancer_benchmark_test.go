package benchmark

import (
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"io/ioutil"
	"log"
	"sync"
	"testing"
	"time"
)

func BenchmarkLoadBalancer(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	time.Sleep(time.Second)

	//create test needed variables
	var waitGroup sync.WaitGroup
	var clientSideLoadBalancer *loadBalancer.LoadBalancerClientSide
	var clientSideServerList []string
	var registrySideLoadBalancer *loadBalancer.LoadBalancerRegistrySide
	var registrySideServerList []string

	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	// create registry for registry-side load balancer
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":8021", registryChannel, &waitGroup)
	testRegistry := <-registryChannel
	waitGroup.Wait()

	//create 2 servers to simulate both client-side and registry-side load balancing
	serverChannelA := make(chan *server.Server)
	serverChannelB := make(chan *server.Server)
	waitGroup.Add(2)
	go createServer(":9021", serviceList, serverChannelA, &waitGroup)
	testServerA := <-serverChannelA
	go createServer(":9022", serviceList, serverChannelB, &waitGroup)
	testServerB := <-serverChannelB
	waitGroup.Wait()

	//servers send heartbeat to the registry to register as alive server to simulate registry-side load balancing
	testServerA.Heartbeat(testRegistry.RegistryURL, 0)
	testServerB.Heartbeat(testRegistry.RegistryURL, 0)

	//create client-side load balancer for testing
	clientSideLoadBalancer = loadBalancer.CreateLoadBalancerClientSide([]string{testServerA.ServerAddress, testServerB.ServerAddress})
	clientSideServerList, _ = clientSideLoadBalancer.GetServerList()

	//create registry-side load balancer for testing
	registrySideLoadBalancer = loadBalancer.CreateLoadBalancerRegistrySide(testRegistry.RegistryURL, 0)
	registrySideServerList, _ = registrySideLoadBalancer.GetServerList()

	b.Run("ClientSide.CreateLoadBalancerClientSide", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = loadBalancer.CreateLoadBalancerClientSide([]string{testServerA.ServerAddress, testServerB.ServerAddress})
		}
	})

	b.Run("ClientSide.GetServer.RandomSelectMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = clientSideLoadBalancer.GetServer(loadBalancer.RandomSelectMode)
		}
	})

	b.Run("ClientSide.GetServer.RoundRobinSelectMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = clientSideLoadBalancer.GetServer(loadBalancer.RoundRobinSelectMode)
		}
	})

	b.Run("ClientSide.GetServerList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = clientSideLoadBalancer.GetServerList()
		}
	})

	// not necessary to test ClientSide.RefreshServerList, only added for test coverage rate
	b.Run("ClientSide.RefreshServerList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clientSideLoadBalancer.RefreshServerList()
		}
	})

	b.Run("ClientSide.UpdateServerList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clientSideLoadBalancer.UpdateServerList(clientSideServerList)
		}
	})

	b.Run("RegistrySide.CreateLoadBalancerRegistrySide", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = loadBalancer.CreateLoadBalancerRegistrySide(testRegistry.RegistryURL, 0)
		}
	})

	b.Run("RegistrySide.GetServer.RandomSelectMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registrySideLoadBalancer.GetServer(loadBalancer.RandomSelectMode)
		}
	})

	b.Run("RegistrySide.GetServer.RoundRobinSelectMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registrySideLoadBalancer.GetServer(loadBalancer.RoundRobinSelectMode)
		}
	})

	b.Run("RegistrySide.GetServerList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registrySideLoadBalancer.GetServerList()
		}
	})

	b.Run("RegistrySide.RefreshServerList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registrySideLoadBalancer.RefreshServerList()
		}
	})

	b.Run("RegistrySide.UpdateServerList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registrySideLoadBalancer.UpdateServerList(registrySideServerList)
		}
	})

}
