package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

/*
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

func (t *Arithmetic) SleepThenAddition(input *Input, output *Output) error {
	time.Sleep(time.Second * time.Duration(input.A))
	output.C = input.A + input.B
	return nil
}
*/

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := server.New_server()
	_ = server.ServerRegister(&foo)
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.AcceptConnection(l)
}


func foo(xc *client.DiscoveryClient, ctx context.Context, typ, serviceMethod string, args *Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

func call(registry string) {
	d := client.NewDiscoveryRegistry(registry, 0)
	xc := client.CreateDiscoveryClient(d, client.RandomSelectMode, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(registry string) {
	d := client.NewDiscoveryRegistry(registry, 0)
	xc := client.CreateDiscoveryClient(d, client.RandomSelectMode, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9999/_rpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	broadcast(registryAddr)
	call(registryAddr)
	//broadcast(registryAddr)
}

/*
func createServer(addressChannel chan string, addressPort string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func createServerHTTP(addressChannel chan string, addressPort string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Println("Server register error:", err)
	}
	listener, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	server.RegisterHandlerHTTP()
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	_ = http.Serve(listener, nil)
	//server.AcceptConnection(listener)
}

func createClientAndCall(address string) {
	testClient, _ := client.MakeDial("tcp", address)
	defer func() { _ = testClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 2 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			clientCallRPC(testClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func createClientAndCallHTTP(addressChannel chan string) {
	testClient, _ := client.MakeDialHTTP("tcp", <-addressChannel)
	defer func() { _ = testClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 2 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			clientCallRPC(testClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func clientCallRPC(client *client.Client, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err := client.Call("Arithmetic.Addition", input, output, timeOutContext); err != nil {
		log.Fatal("Client RPC call Arithmetic.Addition error: ", err)
	}
	log.Printf("%d + %d = %d", input.A, input.B, output.C)
}

func createDiscoveryClientAndCall(addressA string, addressB string) {
	discovery := client.CreateDiscoveryMultiServer([]string{"tcp@" + addressA, "tcp@" + addressB})
	testDiscoveryClient := client.CreateDiscoveryClient(discovery, client.RandomSelectMode, nil)
	defer func() { _ = testDiscoveryClient.Close() }()

	var waitGroup sync.WaitGroup
	n := 0
	for n < 5 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			discoveryClientCallRPC(testDiscoveryClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func createDiscoveryClientAndBroadcast(addressA string, addressB string) {
	discovery := client.CreateDiscoveryMultiServer([]string{"tcp@" + addressA, "tcp@" + addressB})
	testDiscoveryClient := client.CreateDiscoveryClient(discovery, client.RandomSelectMode, nil)
	defer func() { _ = testDiscoveryClient.Close() }()

	var waitGroup sync.WaitGroup
	n := 0
	for n < 5 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			discoveryClientBroadcastRPC(testDiscoveryClient, n)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func discoveryClientCallRPC(discoveryClient *client.DiscoveryClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect no timeout
	err := discoveryClient.Call(context.Background(), "Arithmetic.Addition", input, output)
	if err != nil {
		log.Println("RPC call Arithmetic.Addition error: ", err)
	} else {
		log.Printf("RPC call Arithmetic.Addition success: %d + %d = %d", input.A, input.B, output.C)
	}
}

func discoveryClientBroadcastRPC(discoveryClient *client.DiscoveryClient, number int) {
	input := &Input{A: number, B: number * number}
	output := &Output{}
	//expect 2-5 timeout
	timeOutContext, cancelContext := context.WithTimeout(context.Background(), time.Second*2)
	err := discoveryClient.Broadcast(timeOutContext, "Arithmetic.SleepThenAddition", input, output)
	if err != nil {
		log.Println("RPC call Arithmetic.SleepThenAddition error: ", err)
	} else {
		log.Printf("RPC call Arithmetic.SleepThenAddition success: %d + %d = %d", input.A, input.B, output.C)
	}
	cancelContext()
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//TCP (normal), server address 6666, return 0

	/*	addressChannel := make(chan string)
		go createServer(addressChannel, ":6666")
		address := <-addressChannel
		createClientAndCall(address)
	*/

	//HTTP, server address 7777, not returning (so that debug page is hosted)

	/*	addressChannel := make(chan string)
		go createClientAndCallHTTP(addressChannel)
		createServerHTTP(addressChannel, ":7777")
	*/

	//TCP, load balancing, server address random (:0), return 0

	/*
		addressChannelA := make(chan string)
		addressChannelB := make(chan string)
		go createServer(addressChannelA, ":0")
		go createServer(addressChannelB, ":0")
		addressA := <-addressChannelA
		addressB := <-addressChannelB
		time.Sleep(time.Second)
		createDiscoveryClientAndCall(addressA, addressB)
		createDiscoveryClientAndBroadcast(addressA, addressB)
	*/
}
*/