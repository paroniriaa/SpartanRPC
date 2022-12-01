package main

import (
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

type Input struct {
	A, B int
}

type Output struct {
	C int
}

type Arithmetic int

func (t *Arithmetic) Addition(input *Input, output *Output) error {
	output.C = input.A + input.B
	return nil
}

func (t *Arithmetic) Subtraction(input *Input, output *Output) error {
	output.C = input.A - input.B
	return nil
}

func (t *Arithmetic) Multiplication(input *Input, output *Output) error {
	output.C = input.A * input.B
	return nil
}

func (t *Arithmetic) Division(input *Input, output *Output) error {
	if input.B == 0 {
		return errors.New("divide by zero")
	}
	output.C = input.A / input.B
	return nil
}

func (t *Arithmetic) HeavyComputation(input *Input, output *Output) error {
	time.Sleep(time.Second * time.Duration(input.A))
	rand.Seed(int64(input.B))
	output.C = rand.Int()
	return nil
}

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
		err = testServer.ServiceRegister(service)
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
		err = testServer.ServiceRegister(service)
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
	log.SetFlags(log.Lshortfile)

	//create service type and store it in the service list
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	log.Println("Enter RPC Server Info: [Server_Type] [Machine_Subnet_IP_Address:Port] [Registry_Subnet_IP_Address:Port]")
	var serverType, machineAddressPort, registryAddressPort, registryURL string
	n, err := fmt.Scanln(&serverType, &machineAddressPort, &registryAddressPort)
	if n != 3 {
		log.Println("Initialize RPC Server Info Error: expected 3 arguments: [Server_Type] [Machine_Subnet_IP_Address:Port] [Registry_Subnet_IP_Address:Port]")
	}
	if err != nil {
		log.Fatal(err)
	}

	//registryURL = "http://" + registryAddressPort + registry.DefaultRegistryPath
	if registryAddressPort[:1] == ":" {
		//listener.Addr().String() -> "[::]:1234" -> port extraction needed
		registryURL = "http://localhost" + registryAddressPort + registry.DefaultRegistryPath
	} else {
		//listener.Addr().String() -> "127.0.0.1:1234", port extraction not needed
		registryURL = "http://" + registryAddressPort + registry.DefaultRegistryPath
	}

	if serverType == "HTTP" || serverType == "http" {
		createServerHTTP(machineAddressPort, serviceList, registryURL)
	} else {
		createServer(machineAddressPort, serviceList, registryURL)
	}

}
