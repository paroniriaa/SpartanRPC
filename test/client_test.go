package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"log"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

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
	Output           interface{}
	Expected         interface{}
}

//helper function to create concrete Arithmetic test case
func createArithmeticTestCase(t *testing.T, c *client.Client, ac *ArithmeticCase) {
	t.Helper()
	err := c.Call(ac.ServiceDotMethod, ac.Input, ac.Output)
	es1 := ac.ServiceDotMethod + ":" + " expected no error, but got error %q"
	es2 := ac.ServiceDotMethod + ":" + " %d " + ac.ArithmeticSymbol + " %d " + "expected output %d, but got %d"
	if err != nil {
		t.Errorf(es1, err.Error())
	}
	if ac.Output.C != ac.Expected {
		t.Errorf(es2, ac.Input.A, ac.Input.B, ac.Expected, ac.Output.C)
	}
}

func createBuiltinTypeTestCase(t *testing.T, c *client.Client, btc *BuiltinTypeCase) {
	t.Helper()
	err := c.Call(btc.ServiceDotMethod, btc.Input, btc.Output)
	es1 := btc.ServiceDotMethod + ":" + " expected no error, but got error %q"
	es2 := btc.ServiceDotMethod + ":" + " expected output %v, but got %v"
	if err != nil {
		t.Errorf(es1, err.Error())
	}
	if !reflect.DeepEqual(btc.Output, btc.Expected) {
		t.Errorf(es2, btc.Expected, btc.Output)
	}
}

func createServer(address chan string) {
	var arithmetic Arithmetic
	var builtinType BuiltinType
	err1 := server.ServerRegister(&arithmetic)
	err2 := server.ServerRegister(&builtinType)
	if err1 != nil {
		log.Fatal("Server register error:", err1)
	}
	if err2 != nil {
		log.Fatal("Server register error:", err2)
	}

	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	address <- listener.Addr().String()
	server.Connection_handle(listener)
}

func TestClient(t *testing.T) {
	//create server and client
	address := make(chan string)
	go createServer(address)
	testClient, _ := client.MakeDial("tcp", <-address)
	defer func() { _ = testClient.Close() }()

	//table-driven tests
	arithmeticCases := []ArithmeticCase{
		{"Arithmetic.Addition", "+", &Input{8, 2}, &Output{}, 10},
		{"Arithmetic.Subtraction", "-", &Input{7, 3}, &Output{}, 4},
		{"Arithmetic.Multiplication", "*", &Input{6, 4}, &Output{}, 24},
		{"Arithmetic.Division", "/", &Input{5, 5}, &Output{}, 1},
	}
	input := 7
	var i int
	builtinTypeCases := []BuiltinTypeCase{
		{"BuiltinType.Pointer", "Pointer", input, &i, &input},
		{"BuiltinType.Slice", "Slice", input, &[]int{}, &[]int{input}},
		{"BuiltinType.Array", "Array", input, &[1]int{}, &[1]int{input}},
		{"BuiltinType.Map", "Map", input, &map[int]int{}, &map[int]int{input: input}},
	}

	/*
		log.Printf("BuiltinType.Pointer: %v", builtinTypeCases[0].Output)
		log.Printf("BuiltinType.Slice: %v", builtinTypeCases[1].Output)
		log.Printf("BuiltinType.Array: %v", builtinTypeCases[2].Output)
		log.Printf("BuiltinType.Map: %v", builtinTypeCases[3].Output)
	*/

	//loop through all test arithmeticCases in the table
	//Synchronous calls test
	prefix := "SynchronousCalls."
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			createArithmeticTestCase(t, testClient, &testCase)
		})
	}
	//Asynchronous calls test
	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	prefix = "AsynchronousCalls."
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createArithmeticTestCase(t, testClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()
	//Builtin Type test
	prefix = "TypeCheckingCalls."
	for _, testCase := range builtinTypeCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			createBuiltinTypeTestCase(t, testClient, &testCase)
		})
	}

}
