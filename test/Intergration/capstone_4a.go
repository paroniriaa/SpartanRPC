package main

import (
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"log"
	"sync"
	"time"
)

//Capstone 4A:
//Server: 2 TCP-based RPC server, auto heartbeat sending
//Client: 2 TCP-based RPC client, controlled by 1 RPC client manager
//Load Balancer: 1 TCP-based registry-side load balancer, auto server update, service discover on
//Registry: 1 HTTP-based RPC registry, auto server refresh, heartbeat monitoring on, service dispatch on
//Call: 6 RPC normal calls, 6 RPC broadcast calls (12 normal calls for 2 server), called from RPC client manager via indirect Call()
//Expected result: All 6 RPC normal calls are handled, 2 RPC broadcast calls are handled, 4 RPC broadcast calls are timeout, return 0

func main() {
	//set up logger
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	//use waitGroup for goroutines synchronization
	var waitGroup sync.WaitGroup
	//create service type and store it in the service list
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	time.Sleep(time.Second)
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":0", registryChannel, &waitGroup)
	testRegistry := <-registryChannel
	log.Printf("main -> main: Registry address fetched from registryChannel: %s", testRegistry.RegistryURL)
	waitGroup.Wait()
	serverChannelA := make(chan *server.Server)
	serverChannelB := make(chan *server.Server)
	waitGroup.Add(2)
	go createServer(":0", serviceList, serverChannelA, &waitGroup)
	serverA := <-serverChannelA
	serverA.Heartbeat(testRegistry.RegistryURL, 0)
	log.Printf("main -> main: Server A address fetched from serverChannelA: %s", serverA.ServerAddress)
	go createServer(":0", serviceList, serverChannelB, &waitGroup)
	serverB := <-serverChannelB
	serverB.Heartbeat(testRegistry.RegistryURL, 0)
	log.Printf("main -> main: Server B address fetched from serverChannelB: %s", serverB.ServerAddress)
	waitGroup.Wait()
	createLoadBalancedClientAndCallOnRegistry(testRegistry.RegistryURL)
	createLoadBalancedClientAndBroadcastCallOnRegistry(testRegistry.RegistryURL)
}
