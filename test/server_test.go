package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}


func startServer(address chan string) {
	var foo Foo
	if err := server.ServerRegister(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	address <- l.Addr().String()
	server.AcceptConnection(l)

}

func TestServer(test *testing.T) {
	test.Helper()
	//test.Parallel()

	test.Run("request loop", func(t *testing.T) {
		log.SetFlags(0)
		addr := make(chan string)
		go startServer(addr)
		client, _ := client.MakeDial("tcp", <-addr)
		defer func() { _ = client.Close() }()

		time.Sleep(time.Second)
		// send request & receive response
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				args := &Input{A: i, B: i * i}
				var reply int
				if err := client.Call("Foo.Sum", args, &reply,context.Background()); err != nil {
					log.Fatal("call Foo.Sum error:", err)
				}
			}(i)
		}
		wg.Wait()
	})

	test.Run("TestServerNoParams", func(t *testing.T) {
		addr := make(chan string)
		go startServer(addr)
		conn, _ := net.Dial("tcp", <-addr)
		defer func() { _ = conn.Close() }()

		time.Sleep(time.Second)
		_ = json.NewEncoder(conn).Encode(server.DefaultConnectionInfo)
		cc := coder.NewJsonCoder(conn)
		h := &coder.MessageHeader{
			ServiceDotMethod: "Foo.Sum",
			SequenceNumber:           uint64(1),
		}
		_ = cc.EncodeMessageHeaderAndBody(h, fmt.Sprintf("geerpc req %d", h.SequenceNumber))
		_ = cc.DecodeMessageHeader(h)
		var reply string
		_ = cc.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	})

	test.Run("TestServerEmpty", func(t *testing.T) {
		addr := make(chan string)
		go startServer(addr)
		conn, _ := net.Dial("tcp", <-addr)
		defer func() { _ = conn.Close() }()

		time.Sleep(time.Second)
		_ = json.NewEncoder(conn).Encode(server.DefaultConnectionInfo)
		cc := coder.NewJsonCoder(conn)
		h := &coder.MessageHeader{
			ServiceDotMethod: "",
			SequenceNumber:           uint64(0),
		}
		_ = cc.EncodeMessageHeaderAndBody(h, fmt.Sprintf("geerpc req %d", h.SequenceNumber))
		_ = cc.DecodeMessageHeader(h)
		var reply string
		_ = cc.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	})


}

