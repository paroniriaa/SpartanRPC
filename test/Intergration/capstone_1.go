package main

import (
	"Distributed-RPC-Framework/server"
	"log"
	"sync"
	"time"
)

//Capstone 1:
//Server: 1 TCP-based RPC server
//Client: 1 TCP-based RPC client, connected to RPC server via direct Make-Dial()
//Load Balancer: N/A
//Registry: N/A
//Call: 6 RPC normal calls, called from RPC client via direct Call()
//Expected result: All RPC calls are handled, return 0

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
	serverChannel := make(chan *server.Server)
	waitGroup.Add(1)
	go createServer(":0", serviceList, serverChannel, &waitGroup)
	testServer := <-serverChannel
	log.Printf("main -> main: Server address fetched from addressChannel: %s", testServer.ServerAddress)
	waitGroup.Wait()
	createClientAndCall(testServer.ServerAddress)
}
