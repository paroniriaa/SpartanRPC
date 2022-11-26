package main

import (
	"Distributed-RPC-Framework/server"
	"log"
	"net"
	"sync"
)

func createServerCGo(port string, serviceList []any, serverChannel chan *server.Server, waitGroup *sync.WaitGroup) {
	log.Println("main -> createServer: RPC server initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServerHTTP(listener)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server creation issue:", err)
	}
	for _, service := range serviceList {
		err = testServer.ServerRegister(service)
		if err != nil {
			log.Fatal("main -> createServer error: RPC server register error:", err)
		}
	}
	serverChannel <- testServer
	waitGroup.Done()
	log.Println("main -> createServer: RPC server initialization routine end, now launched and accepting...")
	//BLOCKING and keep listening
	testServer.LaunchAndServe()
}

func createServerC(port string, serviceList []any, registryURL string) {
	log.Println("main -> createServer: RPC server initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServerHTTP(listener)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server creation issue:", err)
	}
	for _, service := range serviceList {
		err = testServer.ServerRegister(service)
		if err != nil {
			log.Fatal("main -> createServer error: RPC server register error:", err)
		}
	}
	log.Printf("main -> createServer: Server A address fetched: %s", testServer.ServerAddress)
	log.Println("main -> createServer: RPC server initialization routine end, now launched and accepting...")
	testServer.Heartbeat(registryURL, 0)
	//BLOCKING and keep listening
	testServer.LaunchAndServe()
}

func main() {
	//set up logger
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	//registryURL =  http://localhost:8001/_srpc_/registry
	registryURL := "http://localhost:8001/_srpc_/registry"

	//create service type and store it in the service list
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	/*	var waitGroup sync.WaitGroup
		time.Sleep(time.Second)
		serverChannelA := make(chan *server.Server)
		waitGroup.Add(1)
		go createServerAGo(":9001", serviceList, serverChannelA, &waitGroup)
		serverA := <-serverChannelA
		log.Printf("main -> main: Server A address fetched from serverChannelA: %s", serverA.ServerAddress)
		waitGroup.Wait()*/

	createServerC(":9003", serviceList, registryURL)
}
