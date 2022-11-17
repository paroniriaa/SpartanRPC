package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"
)

type Input struct {
	A, B int
}

type Output struct {
	C int
}

type Demo int

func (t *Demo) Addition_demo(input Input, output *int) error {
	*output = input.A + input.B
	return nil
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

func (t *Arithmetic) Error(input *Input, output *Output) error {
	panic("ERROR")
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

//helper function to create concrete BuiltinType test case
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

//helper function to create concrete TimeoutCall test case
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

//helper function to create concrete TimeoutDial test case
func createTimeoutDialTestCase(t *testing.T, tc *TimeoutCase) {
	t.Helper()
	fakeClientInitializer := func(cn net.Conn, cni *server.ConnectionInfo) (client *client.Client, err error) {
		_ = cn.Close()
		time.Sleep(time.Second * 2)
		return nil, nil
	}
	_, err := client.MakeDialWithTimeout(fakeClientInitializer, "tcp", tc.Address, tc.ConnectionInfo)
	//log.Println(err)
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

// helper function to create concrete load balancer client-side test case
func createNormalCallTestCase(t *testing.T, c *client.LoadBalancedClient, ac *ArithmeticCase) {
	t.Helper()
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err := c.Call(ctx, ac.ServiceDotMethod, ac.Input, ac.Output)
	if err != nil {
		es1 := ac.ServiceDotMethod + ":" + " expected no error, but got error %q"
		t.Errorf(es1, err.Error())
	}
	if ac.Output.C != ac.Expected {
		es2 := ac.ServiceDotMethod + ":" + " %d " + ac.ArithmeticSymbol + " %d " + "expected output %d, but got %d"
		t.Errorf(es2, ac.Input.A, ac.Input.B, ac.Expected, ac.Output.C)
	}
}

func createBroadcastCallTestCase(t *testing.T, c *client.LoadBalancedClient, ac *ArithmeticCase) {
	t.Helper()
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err := c.BroadcastCall(ctx, ac.ServiceDotMethod, ac.Input, ac.Output)
	if err != nil {
		es1 := ac.ServiceDotMethod + ":" + " expected no error, but got error %q"
		t.Errorf(es1, err.Error())
	}
	if ac.Output.C != ac.Expected {
		es2 := ac.ServiceDotMethod + ":" + " %d " + ac.ArithmeticSymbol + " %d " + "expected output %d, but got %d"
		t.Errorf(es2, ac.Input.A, ac.Input.B, ac.Expected, ac.Output.C)
	}
}

//helper function to create a server with desired services on specified port
func createServer(address chan string, addressPort string, services []any) {
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("RPC server -> createServer error: Server Network issue:", err)
	}
	log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	testServer := server.CreateServer(listener.Addr())
	for _, service := range services {
		err = testServer.ServerRegister(service)
		if err != nil {
			log.Fatal("RPC server -> createServer error: Server register error:", err)
		}
	}
	address <- listener.Addr().String()
	testServer.AcceptConnection(listener)
}

//helper function to create a registry on port 9999
func createRegistry() {
	//log.Println("main -> createRegistry: RPC registry initialization routine start...")
	listener, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	_ = http.Serve(listener, nil)
	//log.Println("main -> createRegistry: RPC registry initialization routine end...")
}

//helper function to create a server with desired services on specified port, and register to the specified registry
func createServerOnRegistry(registryAddress string, addressPort string, services []any) {
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("RPC server -> createServer error: Server Network issue:", err)
	}
	log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	testServer := server.CreateServer(listener.Addr())
	for _, service := range services {
		err = testServer.ServerRegister(service)
		if err != nil {
			log.Fatal("RPC server -> createServer error: Server register error:", err)
		}
	}
	testServer.Heartbeat(registryAddress, 0)
	testServer.AcceptConnection(listener)
}
