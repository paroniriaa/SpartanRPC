package client


import (
	"Distributed-RPC-Framework/server"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

func start(address chan string) {
	portNumber, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("start RPC server on port", portNumber.Addr())
	address <- portNumber.Addr().String()
	server.Connection_handle(portNumber)
}

func TestClient(test *testing.T) {
	test.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	address := make(chan string)
	go start(address)
	client, _ := Connection("tcp", <-address)
	defer func() { _ = client.Close() }()
	time.Sleep(time.Second)
	var waitGroup sync.WaitGroup

	n := 0
	for n < 5 {
		waitGroup.Add(1)
		go func(n int) {
			defer waitGroup.Done()
			args := fmt.Sprintf("geerpc req %d", n)
			var reply string
			if err := client.Call("Foo.Sum",args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(n)
		n++
	}
	waitGroup.Wait()
}

