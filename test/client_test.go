package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"context"
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

//helper function to create concrete Arithmetic test case
func createArithmeticTestCase(t *testing.T, c *client.Client, ac *ArithmeticCase) {
	t.Helper()
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err := c.Call(ac.ServiceDotMethod, ac.Input, ac.Output, ctx)
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
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err := c.Call(btc.ServiceDotMethod, btc.Input, btc.Output, ctx)
	es1 := btc.ServiceDotMethod + ":" + " expected no error, but got error %q"
	es2 := btc.ServiceDotMethod + ":" + " expected output %v, but got %v"
	if err != nil {
		t.Errorf(es1, err.Error())
	}
	if !reflect.DeepEqual(btc.Output, btc.Expected) {
		t.Errorf(es2, btc.Expected, btc.Output)
	}
}

func createTimeoutCallTestCase(t *testing.T, tc *TimeoutCase) {
	t.Helper()
	c, _ := client.MakeDial("tcp", tc.Address, tc.ConnectionInfo)
	defer func() { _ = c.Close() }()
	err := c.Call(tc.ServiceDotMethod, tc.Input, tc.Output, tc.Context)
	log.Println(err)
	es1 := tc.ServiceDotMethod + ":" + " expected" + tc.TimeoutType + " timeout error but got nil error"
	if err == nil {
		t.Errorf(es1)
	}
}

func createTimeoutDialTestCase(t *testing.T, tc *TimeoutCase) {
	t.Helper()
	fakeClientInitializer := func(cn net.Conn, cni *server.ConnectionInfo) (client *client.Client, err error) {
		_ = cn.Close()
		time.Sleep(time.Second * 2)
		return nil, nil
	}
	_, err := client.MakeDialWithTimeout(fakeClientInitializer, "tcp", tc.Address, tc.ConnectionInfo)
	log.Println(err)
	if tc.ConnectionInfo.ConnectionTimeout == 0 {
		if err != nil {
			es := tc.ServiceDotMethod + tc.TimeoutType + ":" + " expected nil timeout error but got error"
			t.Errorf(es)
		}
	} else {
		if err == nil {
			es := tc.ServiceDotMethod + tc.TimeoutType + ":" + " expected timeout error but got nil error"
			t.Errorf(es)
		}
	}
}

func createServer(address chan string, services []any) {
	for _, service := range services {
		registerService(service)
	}

	listener, err := net.Listen("tcp", "localhost:8001")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	address <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func registerService(service any) {
	err := server.ServerRegister(service)
	if err != nil {
		log.Fatal("Server register error:", err)
	}
}

func TestClient_Call(t *testing.T) {
	t.Helper()
	//create service type
	var arithmetic Arithmetic
	var builtinType BuiltinType
	var timeOut Timeout
	services := []any{
		&arithmetic,
		&builtinType,
		&timeOut,
	}
	addressChannel := make(chan string)
	go createServer(addressChannel, services)
	address := <-addressChannel
	defaultClient, _ := client.MakeDial("tcp", address)
	//defer func() { _ = defaultClient.Close() }()

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

	//Set 1 sec as timeout limit on the client-side (after connection)
	//Range: from client sending the request to server, to client receiving the response from server
	//expected to fail because Timeout.SleepForTimeout takes 5 secs, and client-side time limit is 1 sec
	smallTimeoutContext, _ := context.WithTimeout(context.Background(), time.Second)
	//Set 1 sec as timeout limit on the server-side (after connection)
	//Range: from server sending the request to service(concrete method), to server receiving the response from service
	//expected to fail because Timeout.SleepForTimeout takes 5 secs, and server-side time limit is 1 sec
	noTimeoutContext := context.Background()
	shortProcessTimeoutConnectionInfo := &server.ConnectionInfo{ProcessingTimeout: time.Second}
	timeoutCallCases := []TimeoutCase{
		{"Timeout.SleepForTimeout", "Handling.Client-Side", input, &i, address, smallTimeoutContext, nil},
		{"Timeout.SleepForTimeout", "Handling.Server-Side", input, &i, address, noTimeoutContext, shortProcessTimeoutConnectionInfo},
	}
	//Set 1 sec as timeout limit on the client-side (before connection)
	//Range: from initializing a client, to returning a connected client
	//expected to fail because fakeClientInitializer takes 5 secs, and ConnectionTimeout is 1 sec
	shortConnectionTimeoutConnectionInfo := &server.ConnectionInfo{ConnectionTimeout: time.Second}
	//Set 1 sec as timeout limit on the client-side (before connection)
	//Range: from initializing a client, to returning a connected client
	//expected to pass because ConnectionTimeout is 0, which means there is no timeout limit
	noConnectionTimeoutConnectionInfo := &server.ConnectionInfo{ConnectionTimeout: 0}
	timeoutDialCases := []TimeoutCase{
		{"MakeDialWithTimeout", "Connection.1 Sec Limit", input, &i, address, nil, shortConnectionTimeoutConnectionInfo},
		{"MakeDialWithTimeout", "Connection.No Limit", input, &i, address, nil, noConnectionTimeoutConnectionInfo},
	}

	//Synchronous calls tests
	prefix := "SynchronousCalls."
	time.Sleep(time.Second)
	//loop through all arithmeticCases in the table and create the concrete subtest
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			createArithmeticTestCase(t, defaultClient, &testCase)
		})
	}

	//Asynchronous calls test
	prefix = "AsynchronousCalls."
	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	//loop through all arithmeticCases in the table and create the concrete subtest
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			waitGroup.Add(1)
			go func(testCase ArithmeticCase) {
				defer waitGroup.Done()
				createArithmeticTestCase(t, defaultClient, &testCase)
			}(testCase)
		})
	}
	waitGroup.Wait()

	//Builtin type calls test
	prefix = "TypeCheckingCalls."
	time.Sleep(time.Second)
	//loop through all builtinTypeCases in the table and create the concrete subtest
	for _, testCase := range builtinTypeCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			createBuiltinTypeTestCase(t, defaultClient, &testCase)
		})
	}

	_ = defaultClient.Close()

	//Timeout calls test
	prefix = "TimeoutCheckingCalls."
	time.Sleep(time.Second)
	//loop through all timeoutCallCases in the table and create the concrete subtest
	for _, testCase := range timeoutCallCases {
		t.Run(prefix+testCase.TimeoutType, func(t *testing.T) {
			createTimeoutCallTestCase(t, &testCase)
		})
	}

	//Timeout dials test
	prefix = "TimeoutCheckingDials."
	time.Sleep(time.Second)
	//loop through all timeoutDialCases in the table and create the concrete subtest
	for _, testCase := range timeoutDialCases {
		t.Run(prefix+testCase.TimeoutType, func(t *testing.T) {
			createTimeoutDialTestCase(t, &testCase)
		})
	}
}
