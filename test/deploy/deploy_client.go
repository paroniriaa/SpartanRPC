package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"context"
	"fmt"
	"log"
	"os"
)

func createClient(registryURL string) {
	var testCase ArithmeticCase

	registryLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(registryURL, 0)
	loadBalancedClient := client.CreateLoadBalancedClient(registryLoadBalancer, loadBalancer.RoundRobinSelectMode, nil)
	defer func() { _ = loadBalancedClient.Close() }()
	log.Printf("Before Call -> clientLoadBalancer: %+v", registryLoadBalancer)
	log.Printf("Before Call -> loadBalancedClient: %+v", loadBalancedClient)

	//log.Println("sRPC Call example: Arithmetic.Addition 1 1")
	for {
		log.Println("Enter sRPC Call info: [Service.Method] [NumberA] [NumberB]")
		var serviceDotMethod, arithmeticSymbol string
		var numberA, numberB int
		n, err := fmt.Scanln(&serviceDotMethod, &numberA, &numberB)
		if serviceDotMethod == "exit" {
			os.Exit(0)
		}
		if n != 3 {
			log.Println("Initialize sRPC Call error: expected 3 arguments: [Service.Method] [NumberA] [NumberB]")
		}
		if err != nil {
			log.Fatal(err)
		}
		switch serviceDotMethod {
		case "Arithmetic.Addition":
			arithmeticSymbol = "+"
			break
		case "Arithmetic.Subtraction":
			arithmeticSymbol = "-"
			break
		case "Arithmetic.Multiplication":
			arithmeticSymbol = "*"
			break
		case "Arithmetic.Division":
			arithmeticSymbol = "/"
			break
		case "A.A":
			serviceDotMethod = "Arithmetic.Addition"
			arithmeticSymbol = "+"
			break
		case "A.S":
			serviceDotMethod = "Arithmetic.Subtraction"
			arithmeticSymbol = "-"
			break
		case "A.M":
			serviceDotMethod = "Arithmetic.Multiplication"
			arithmeticSymbol = "*"
			break
		case "A.D":
			serviceDotMethod = "Arithmetic.Division"
			arithmeticSymbol = "/"
			break
		default:
			log.Printf("Initialize sRPC Call error sRPC Call failed: [Service.Method] %s does not exist, plesae retry...", serviceDotMethod)
			continue

		}
		log.Printf("Invoking sRPC Call: %s -> %d %s %d", serviceDotMethod, numberA, arithmeticSymbol, numberB)
		testCase = ArithmeticCase{
			serviceDotMethod,
			arithmeticSymbol,
			&Input{numberA, numberB},
			&Output{},
			0,
		}
		//expect no timeout
		err = loadBalancedClient.Call(context.Background(), testCase.ServiceDotMethod, testCase.Input, testCase.Output)
		if err != nil {
			log.Printf("sRPC call %s error: %s", testCase.ServiceDotMethod, err)
		} else {
			log.Printf("sRPC call %s success: %d %s %d = %d", testCase.ServiceDotMethod, testCase.Input.A, arithmeticSymbol, testCase.Input.B, testCase.Output.C)
		}
	}
}

func main() {
	//set up logger
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	log.Println("Enter RPC Client Info: [Registry_Subnet_IP_Address:Port]")
	var registryAddressPort, registryURL string
	n, err := fmt.Scanln(&registryAddressPort)
	if n != 1 {
		log.Println("Initialize RPC Client Info error: expected 1 argument1: [Registry_Subnet_IP_Address:Port]")
	}
	if err != nil {
		log.Fatal(err)
	}

	registryURL = "http://" + registryAddressPort + registry.DefaultPath

	createClient(registryURL)

}
