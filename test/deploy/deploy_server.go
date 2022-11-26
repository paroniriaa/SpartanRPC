package main

import (
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"fmt"
	"log"
	"net"
)

func createServer(addressPort string, serviceList []any, registryURL string) {
	log.Println("main -> createServer: RPC server initialization routine start...")
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServer(listener)
	if err != nil {
		log.Fatal("main -> createServer error: RPC server creation issue:", err)
	}
	for _, service := range serviceList {
		err = testServer.ServerRegister(service)
		if err != nil {
			log.Fatal("main -> createServer error: RPC server register error:", err)
		}
	}
	log.Printf("main -> createServer: Server address fetched: %s", testServer.ServerAddress)
	log.Println("main -> createServer: RPC server initialization routine end, now launched and accepting...")
	testServer.Heartbeat(registryURL, 0)
	//BLOCKING and keep listening TCP
	testServer.LaunchAndAccept()
}

func createServerHTTP(addressPort string, serviceList []any, registryURL string) {
	log.Println("main -> createServerHTTP: RPC HTTP server initialization routine start...")
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServerHTTP(listener)
	if err != nil {
		log.Fatal("main -> createServerHTTP error: RPC server creation issue:", err)
	}
	for _, service := range serviceList {
		err = testServer.ServerRegister(service)
		if err != nil {
			log.Fatal("main -> createServerHTTP error: RPC server register error:", err)
		}
	}
	log.Printf("main -> createServerHTTP: Server address fetched: %s", testServer.ServerAddress)
	log.Println("main -> createServerHTTP: RPC HTTP server initialization routine end, now launched and accepting...")
	testServer.Heartbeat(registryURL, 0)
	//BLOCKING and keep listening HTTP
	testServer.LaunchAndServe()
}

func main() {
	//set up logger
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//create service type and store it in the service list
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	log.Println("Enter RPC Server Info: [Server_Type] [Machine_Subnet_IP_Address:Port] [Registry_Subnet_IP_Address:Port]")
	var serverType, machineAddressPort, registryAddressPort, registryURL string
	n, err := fmt.Scanln(&serverType, &machineAddressPort, &registryAddressPort)
	if n != 3 {
		log.Println("Initialize RPC Server Info error: expected 3 arguments: [Server_Type] [Machine_Subnet_IP_Address:Port] [Registry_Subnet_IP_Address:Port]")
	}
	if err != nil {
		log.Fatal(err)
	}
	registryURL = "http://" + registryAddressPort + registry.DefaultPath
	if serverType == "HTTP" || serverType == "http" {
		createServerHTTP(machineAddressPort, serviceList, registryURL)
	} else {
		createServer(machineAddressPort, serviceList, registryURL)
	}

}
