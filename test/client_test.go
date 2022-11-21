package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"context"
	"log"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	var waitGroup sync.WaitGroup
	//create service type
	var arithmetic Arithmetic
	var builtinType BuiltinType
	var timeOut Timeout
	services := []any{
		&arithmetic,
		&builtinType,
		&timeOut,
	}

	//create server for testing
	addressChannel := make(chan string)
	waitGroup.Add(1)
	go createServer(":0", services, addressChannel, &waitGroup)
	address := <-addressChannel
	waitGroup.Wait()
	//create client for testing
	defaultClient, _ := client.MakeDial("tcp", address)
	defer func() { _ = defaultClient.Close() }()

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
	//time.Sleep(time.Second)
	//loop through all arithmeticCases in the table and create the concrete subtest
	for _, testCase := range arithmeticCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			createArithmeticTestCase(t, defaultClient, &testCase)
		})
	}

	//Asynchronous calls test
	prefix = "AsynchronousCalls."
	//time.Sleep(time.Second)
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
	//time.Sleep(time.Second)
	//loop through all builtinTypeCases in the table and create the concrete subtest
	for _, testCase := range builtinTypeCases {
		t.Run(prefix+testCase.ServiceDotMethod, func(t *testing.T) {
			createBuiltinTypeTestCase(t, defaultClient, &testCase)
		})
	}

	_ = defaultClient.Close()

	//Timeout calls test
	prefix = "TimeoutCheckingCalls."
	//time.Sleep(time.Second)
	//loop through all timeoutCallCases in the table and create the concrete subtest
	for _, testCase := range timeoutCallCases {
		t.Run(prefix+testCase.TimeoutType, func(t *testing.T) {
			createTimeoutCallTestCase(t, &testCase)
		})
	}

	//Timeout dials test
	prefix = "TimeoutCheckingDials."
	//time.Sleep(time.Second)
	//loop through all timeoutDialCases in the table and create the concrete subtest
	for _, testCase := range timeoutDialCases {
		t.Run(prefix+testCase.TimeoutType, func(t *testing.T) {
			createTimeoutDialTestCase(t, &testCase)
		})
	}

	//client direct HTTP XDials test (windows)
	t.Run("WindowsXDial", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			windowsAddressChannel := make(chan string)
			go func() {
				listener, err := net.Listen("tcp", ":0")
				if err != nil {
					t.Error("failed to listen windows socket")
					return
				}
				testHTTPServer, err := server.CreateServer(listener)
				windowsAddressChannel <- listener.Addr().String()
				testHTTPServer.LaunchAndServe()
			}()
			serverAddress := <-windowsAddressChannel
			_, err := client.XMakeDial("http@" + serverAddress)
			_assert(err == nil, "failed to connect windows socket", err)
		} else {
			log.Println("current GO OS is not windows, corresponding sub tests has been dumped")
		}
	})

	//client direct HTTP XDials test (Linux)
	t.Run("LinuxXDial", func(t *testing.T) {
		if runtime.GOOS == "linux" {
			linuxAddressChannel := make(chan string)
			go func() {
				listener, err := net.Listen("unix", ":0")
				if err != nil {
					t.Error("failed to listen unix socket")
					return
				}
				testHTTPServer, err := server.CreateServer(listener)
				linuxAddressChannel <- listener.Addr().String()
				testHTTPServer.LaunchAndServe()
			}()
			serverAddress := <-linuxAddressChannel
			_, err := client.XMakeDial("unix@" + serverAddress)
			_assert(err == nil, "failed to connect unix socket", err)
		} else {
			log.Println("current GO OS is not linux, corresponding sub tests has been dumped")
		}
	})
}
