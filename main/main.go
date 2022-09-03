package main

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"context"
	"log"
	"net"
	"net/http"
	"sync"
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

func (t Arithmetic) Sleep(input Input, output *int) error {
	time.Sleep(time.Second * time.Duration(input.A))
	*output = input.A + input.B
	return nil
}


func createServer(addressChannel chan string) {

	var arithmetic Arithmetic

	listener, _ := net.Listen("tcp",":0")
	server := server.New_server()
	_ = server.ServerRegister(&arithmetic)
	addressChannel <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func arithmetic(cli *xclient.XClient, message context.Context, callingMethod, serviceName string, input *Input) {
	var output int
	var err error

	if callingMethod == "call"{
		err = cli.Call(message, serviceName, input, &output)
	} else if callingMethod == "broadcast"{
		err = cli.Broadcast(message, serviceName, input, &output)
	} else {
		log.Printf("callingMethod error: %s does not exist", callingMethod)
	}

	if err != nil {
		log.Printf("Calling method:%s \n serviceName: %s \n status: fail  \n   reason: %v", callingMethod, serviceName, err)
	} else {
		log.Printf("Calling method:%s \n serviceName: %s \n status:success: \n process: %d + %d = %d", callingMethod, serviceName, input.A, input.B, output)
	}
}

func calling(address1, address2 string) {
	discoverService := xclient.NewMultiServerDiscovery([]string{"tcp@" + address1, "tcp@" + address2})
	client := xclient.NewXClient(discoverService, xclient.RandomSelect, nil)
	defer func() { _ = client.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			arithmetic(xc, context.Background(), "call", "Arithmetic.Addition", &Input{A: i, B: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
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
	ch1 := make(chan string)
	ch2 := make(chan string)
	// start two servers
	go createServer(ch1)
	go createServer(ch2)

	addr1 := <-ch1
	addr2 := <-ch2

	time.Sleep(time.Second)
	calling(addr1, addr2)
	broadcast(addr1, addr2)
}

/*
func createServer(addressChannel chan string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("Server register error:", err)
	}
	listener, err := net.Listen("tcp", ":6666")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func createServerHTTP(addressChannel chan string) {
	var arithmetic Arithmetic
	err := server.ServerRegister(&arithmetic)
	if err != nil {
		log.Fatal("Server register error:", err)
	}
	listener, err := net.Listen("tcp", ":7777")
	if err != nil {
		log.Fatal("Server Network issue:", err)
	}
	server.RegisterHandlerHTTP()
	//log.Println("RPC server -> createServer: RPC server created and hosting on port", listener.Addr())
	addressChannel <- listener.Addr().String()
	_ = http.Serve(listener, nil)
	//server.AcceptConnection(listener)
}







func createClientAndCall(addressChannel chan string) {
	testClient, _ := client.MakeDial("tcp", <-addressChannel)
	defer func() { _ = testClient.Close() }()

	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup
	n := 0
	for n < 2 {
		waitGroup.Add(1)
		go func(n int) {
			clientCallRPC(testClient, n, &waitGroup)
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
		//
		go func(n int) {
			clientCallRPC(testClient, n, &waitGroup)
		}(n)
		n++
	}
	waitGroup.Wait()
}

func clientCallRPC(client *client.Client, number int, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	input := &Input{A: number, B: number ^ 2}
	output := &Output{}
	timeOutContext, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err := client.Call("Arithmetic.Addition", input, output, timeOutContext); err != nil {
		log.Fatal("Client RPC call Demo.Sum error:", err)
	}
	log.Printf("%d + %d = %d", input.A, input.B, output.C)
}


func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	addressChannel := make(chan string)

	//TCP (normal), server address 6666
	//go createServer(addressChannel)
	//createClientAndCall(addressChannel)

	//HTTP, server address 7777
	go createClientAndCallHTTP(addressChannel)
	createServerHTTP(addressChannel)
}
*/