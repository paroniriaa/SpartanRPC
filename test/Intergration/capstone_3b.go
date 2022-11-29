package main

import (
	"Distributed-RPC-Framework/server"
	"log"
	"sync"
	"time"
)

//Capstone_1 3B:
//Server: 2 HTTP-based RPC server
//Client: 2 HTTP-based RPC client, controlled by 1 RPC client manager
//Load Balancer: 1 HTTP-based client-side load balancer, manual server update, no registry discover
//Registry: N/A
//Call: 6 RPC normal calls, 6 RPC broadcast calls (12 normal calls for 2 server), called from RPC client manager via indirect Call()
//Expected result: All 6 RPC normal calls are handled, 2 RPC broadcast calls are handled, 4 RPC broadcast calls are timeout, not returning (so that server info page will continue to host)

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
	serverChannelA := make(chan *server.Server)
	serverChannelB := make(chan *server.Server)
	waitGroup.Add(2)
	go createServerHTTP(":0", serviceList, serverChannelA, &waitGroup)
	serverA := <-serverChannelA
	log.Printf("main -> main: Server A address fetched from serverChannelA: %s", serverA.ServerAddress)
	go createServerHTTP(":0", serviceList, serverChannelB, &waitGroup)
	serverB := <-serverChannelB
	waitGroup.Wait()
	log.Printf("main -> main: Server B address fetched from serverChannelB: %s", serverB.ServerAddress)
	createLoadBalancedClientAndCall(serverA.ServerAddress, serverB.ServerAddress)
	createLoadBalancedClientAndBroadcastCall(serverA.ServerAddress, serverB.ServerAddress)

	//create infinite loop to stop main process from terminating to monitor all other go routines
	for {

	}
}
