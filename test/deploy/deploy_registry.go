package main

import (
	"Distributed-RPC-Framework/registry"
	"log"
	"net"
	"sync"
)

func createRegistryGo(port string, registryChannel chan *registry.Registry, waitGroup *sync.WaitGroup) {
	log.Println("main -> createRegistry: RPC registry initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createRegistry error: RPC registry network issue:", err)
	}
	testRegistry, err := registry.CreateRegistry(listener, registry.DefaultTimeout)
	if err != nil {
		log.Fatal("main -> createServer error: RPC registry creation issue:", err)
	}
	registryChannel <- testRegistry
	waitGroup.Done()
	log.Println("main -> createRegistry: RPC registry initialization routine end, now launched and serving...")
	//BLOCKING and keep serving
	testRegistry.LaunchAndServe()
}

func createRegistry(port string) {
	log.Println("main -> createRegistry: RPC registry initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("main -> createRegistry error: RPC registry network issue:", err)
	}
	testRegistry, err := registry.CreateRegistry(listener, registry.DefaultTimeout)
	if err != nil {
		log.Fatal("main -> createServer error: RPC registry creation issue:", err)
	}
	log.Printf("main -> createRegistry: Registry address fetched: %s", testRegistry.RegistryURL)
	log.Println("main -> createRegistry: RPC registry initialization routine end, now launched and serving...")
	//BLOCKING and keep serving
	testRegistry.LaunchAndServe()
}

func main() {
	//set up logger
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	/*
		//use waitGroup for goroutines synchronization
		var waitGroup sync.WaitGroup
		time.Sleep(time.Second)
		registryChannel := make(chan *registry.Registry)
		waitGroup.Add(1)
		go createRegistryGo(":8001", registryChannel, &waitGroup)
		testRegistry := <-registryChannel
		log.Printf("main -> main: Registry address fetched from registryChannel: %s", testRegistry.RegistryURL)*/

	createRegistry(":8001")
}
