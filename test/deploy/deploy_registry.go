package main

import (
	"Distributed-RPC-Framework/registry"
	"fmt"
	"log"
	"net"
)

func createRegistry(addressPort string) {
	log.Println("main -> createRegistry: RPC registry initialization routine start...")
	listener, err := net.Listen("tcp", addressPort)
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
	log.SetFlags(log.Lshortfile)

	log.Println("Enter RPC Registry Info: [Machine_Subnet_IP_Address:Port]")
	var registryAddressPort string
	n, err := fmt.Scanln(&registryAddressPort)
	if n != 1 {
		log.Println("Initialize RPC Registry Info Error: expected 1 argument: [Machine_Subnet_IP_Address:Port]")
	}
	if err != nil {
		log.Fatal(err)
	}

	createRegistry(registryAddressPort)
}
