package main

import (
	"Distributed-RPC-Framework/server"
	"log"
	"sync"
	"time"
)

//Capstone 2:
//Server: 1 HTTP-based RPC server
//Client: 1 HTTP-based RPC client, connected to RPC server via direct Make-Dial-HTTP()
//Load Balancer: N/A
//Registry: N/A
//Call: 6 RPC normal calls, called from RPC client via direct Call()
//Expected result: All RPC calls are handled, not returning (so that server info page will continue to host)

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
	serverChannelHTTP := make(chan *server.Server)
	go createClientAndCallHTTP(serverChannelHTTP)
	waitGroup.Add(1)
	createServerHTTP(":0", serviceList, serverChannelHTTP, &waitGroup)
	waitGroup.Wait()
}
