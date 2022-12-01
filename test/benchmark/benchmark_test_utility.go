package benchmark

import (
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

type Input struct {
	A, B int
}

type Output struct {
	C int
}

type Test int

func (t *Test) Echo(input string, output *string) error {
	*output = input
	return nil
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

type BuiltinType struct{}

func (t *BuiltinType) Pointer(input int, output *int) error {
	*output = input
	return nil
}

func (t *BuiltinType) Slice(input int, output *[]int) error {
	*output = append(*output, input)
	return nil
}

func (t *BuiltinType) Array(input int, output *[1]int) error {
	(*output)[0] = input
	return nil
}

func (t *BuiltinType) Map(input int, output *map[int]int) error {
	(*output)[input] = input
	return nil
}

type Timeout time.Duration

func (p Timeout) SleepForTimeout(input int, output *int) error {
	time.Sleep(time.Second * 5)
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		log.Fatalf("assertion failed: "+msg, v...)
	}
}

type ArithmeticCase struct {
	ServiceDotMethod string
	ArithmeticSymbol string
	Input            *Input
	Output           *Output
	Expected         int
}

type BuiltinTypeCase struct {
	ServiceDotMethod string
	BuiltinType      string
	Input            int
	Output           any
	Expected         any
}

type TimeoutCase struct {
	ServiceDotMethod string
	TimeoutType      string
	Input            int
	Output           any
	Address          string
	Context          context.Context
	ConnectionInfo   *server.ConnectionInfo
}

//helper function to create an RPC server with desired services on specified port
//and have the created RPC server send to the server channel
func createServer(port string, serviceList []any, serverChannel chan *server.Server, waitGroup *sync.WaitGroup) {
	log.Println("Test utility -> createServer: RPC server initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("Test utility -> createServer error: RPC server network issue:", err)
	}
	testServer, err := server.CreateServer(listener)
	if err != nil {
		log.Fatal("Test utility -> createServer error: RPC server creation issue:", err)
	}
	for _, service := range serviceList {
		err = testServer.ServiceRegister(service)
		if err != nil {
			log.Fatal("Test utility -> createServer error: RPC server register error:", err)
		}
	}
	serverChannel <- testServer
	waitGroup.Done()
	log.Println("Test utility -> createServer: RPC server initialization routine end, now launched and accepting...")
	//BLOCKING and keep listening
	testServer.LaunchAndAccept()
}

//helper function to create an RPC HTTP server with desired services on specified port
//and have the created RPC HTTP server send to the server channel
func createServerHTTP(port string, serviceList []any, serverChannelHTTP chan *server.Server, waitGroup *sync.WaitGroup) {
	log.Println("Test utility -> createServerHTTP: RPC HTTP server initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("Test utility -> createServerHTTP error: RPC HTTP server network issue: ", err)
	}
	testServerHTTP, err := server.CreateServerHTTP(listener)
	if err != nil {
		log.Fatal("Test utility -> createServerHTTP error: RPC HTTP server creation issue: ", err)
	}
	for _, service := range serviceList {
		err = testServerHTTP.ServiceRegister(service)
		if err != nil {
			log.Fatal("Test utility -> createServerHTTP error: RPC HTTP server register error:", err)
		}
	}
	serverChannelHTTP <- testServerHTTP
	waitGroup.Done()
	log.Println("Test utility -> createServerHTTP: RPC HTTP server initialization routine end, now launched and serving...")
	//BLOCKING and keep serving
	testServerHTTP.LaunchAndServe()
}

//helper function to create an RPC registry on specified port
//and have the created RPC registry send to the registry channel
func createRegistry(port string, registryChannel chan *registry.Registry, waitGroup *sync.WaitGroup) {
	log.Println("Test utility -> createRegistry: RPC registry initialization routine start...")
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("Test utility -> createRegistry error: RPC registry network issue:", err)
	}
	testRegistry, err := registry.CreateRegistry(listener, registry.DefaultTimeout)
	if err != nil {
		log.Fatal("Test utility -> createRegistry error: RPC registry creation issue:", err)
	}
	registryChannel <- testRegistry
	waitGroup.Done()
	log.Println("Test utility -> createRegistry: RPC registry initialization routine end, now launched and serving...")
	//BLOCKING and keep serving
	testRegistry.LaunchAndServe()
}
