package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

func createClient(registryURL string, connectionInfo *server.ConnectionInfo) {
	var testCase ArithmeticCase

	//log.Printf("connectionInfo: %+v", connectionInfo)
	registryLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(registryURL, 0)
	loadBalancedClient := client.CreateLoadBalancedClient(registryLoadBalancer, loadBalancer.RoundRobinSelectMode, connectionInfo)
	defer func() { _ = loadBalancedClient.Close() }()

	//log.Println("sRPC Call example: Arithmetic.Addition 1 1")
	for {
		log.Printf("Connection Configuration Info: %+v", connectionInfo)
		log.Println("Available RPC service.method: \nA.A -> Arithmetic.Addition \nA.S -> Arithmetic.Subtraction \nA.M -> Arithmetic.Multiplication \nA.D -> Arithmetic.Division \nA.HC -> Arithmetic.HeavyComputation")
		log.Println("Available RPC call type: \nNC -> RPC Normal CALL \nBC -> RPC Broadcast")
		log.Println("Enter sRPC call info: [RPC_Call_Type] [Service.Method] [NumberA] [NumberB]")
		var rpcCallType, serviceDotMethod, arithmeticSymbol string
		var numberA, numberB int
		n, err := fmt.Scanln(&rpcCallType, &serviceDotMethod, &numberA, &numberB)
		if serviceDotMethod == "exit" {
			os.Exit(0)
		}
		if n != 4 {
			log.Println("Initialize sRPC call error: expected 4 arguments: [RPC_Call_Type] [Service.Method] [NumberA] [NumberB]")
		}
		if err != nil {
			log.Fatal(err)
		}
		switch serviceDotMethod {
		case "A.A":
			serviceDotMethod, arithmeticSymbol = "Arithmetic.Addition", "+"
			break
		case "A.S":
			serviceDotMethod, arithmeticSymbol = "Arithmetic.Subtraction", "-"
			break
		case "A.M":
			serviceDotMethod, arithmeticSymbol = "Arithmetic.Multiplication", "*"
			break
		case "A.D":
			serviceDotMethod, arithmeticSymbol = "Arithmetic.Division", "/"
			break
		case "A.HC":
			serviceDotMethod, arithmeticSymbol = "Arithmetic.HeavyComputation", "+"
			break
		default:
			log.Printf("Initialize sRPC call error: [Service.Method] %s does not exist, plesae choose the existing RPC service.method...", serviceDotMethod)
			continue

		}
		testCase = ArithmeticCase{
			serviceDotMethod,
			arithmeticSymbol,
			&Input{numberA, numberB},
			&Output{},
			0,
		}

		switch rpcCallType {
		case "NC":
			rpcCallType = "call"
			err = loadBalancedClient.Call(context.Background(), testCase.ServiceDotMethod, testCase.Input, testCase.Output)
			break
		case "BC":
			rpcCallType = "broadcast"
			err = loadBalancedClient.BroadcastCall(context.Background(), testCase.ServiceDotMethod, testCase.Input, testCase.Output)
			break
		default:
			log.Printf("Initialize sRPC call error: [RPC_Call_Type] %s does not exist, plesae choose the existing RPC call type...", rpcCallType)
			continue
		}
		//err = loadBalancedClient.Call(context.Background(), testCase.ServiceDotMethod, testCase.Input, testCase.Output)
		if err != nil {
			log.Printf("sRPC %s %s error: %s", rpcCallType, testCase.ServiceDotMethod, err)
		} else {
			log.Printf("sRPC %s %s success -> %d %s %d = %d", rpcCallType, testCase.ServiceDotMethod, testCase.Input.A, arithmeticSymbol, testCase.Input.B, testCase.Output.C)
		}
	}
}

func main() {
	//set up logger
	log.SetFlags(log.Lshortfile)

	log.Println("Enter RPC Client Info: [Registry_Subnet_IP_Address:Port] [Connection_Timeout] [Processing_Timeout]")
	var registryAddressPort, registryURL string
	var connectionTimeout, processingTimeout int
	n, err := fmt.Scanln(&registryAddressPort, &connectionTimeout, &processingTimeout)
	if n != 3 {
		log.Println("Initialize RPC Client Info error: expected 3 argument1: [Registry_Subnet_IP_Address:Port] [Connection_Timeout] [Processing_Timeout]")
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

	var connectionInfo = &server.ConnectionInfo{
		IDNumber:          server.MagicNumber,
		CoderType:         coder.Json,
		ConnectionTimeout: time.Second * time.Duration(connectionTimeout),
		ProcessingTimeout: time.Second * time.Duration(processingTimeout),
	}

	createClient(registryURL, connectionInfo)

}
