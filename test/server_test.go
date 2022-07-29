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
	"testing"
	"time"
)

type Method1 int

type Method2 int

type Innerput struct{ A, B int }

func (f Method1) TestSum(input Innerput, output *int) error {
	*output = input.A + input.B
	return nil
}

func (f Method2) TestEcho(input Innerput, output *int) error {
	*output = input.A - input.B
	return nil
}


func startTestServer(testMethod int, address chan string) {
	var me1 Method1
	var me2 Method2
	if testMethod == 1 {
		if err := server.ServerRegister(&me1); err != nil {
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
	} else {
		if err := server.ServerRegister(&me2); err != nil {
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


}

func TestServer_client_connection(test *testing.T) {
	test.Helper()
	//test.Parallel()
	test.Run("request loop", func(t *testing.T) {
		log.SetFlags(0)
		addr := make(chan string)
		go startTestServer(1,addr)
		client, _ := client.MakeDial("tcp", <-addr)
		defer func() { _ = client.Close() }()

		time.Sleep(time.Second)
		// send request & receive response
		input := &Innerput{A: 1, B: 2 * 3}
		var output int
		if err := client.Call("Method1.TestSum", input, &output,context.Background()); err != nil {
			log.Fatal("call Foo.Sum error:", err)
		}
	})
}

func TestServer_argv(test *testing.T) {
	test.Helper()
	//test.Parallel()
	address := make(chan string)
	go startTestServer(2,address)
	socket, _ := net.Dial("tcp", <-address)
	defer func() { _ = socket.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(socket).Encode(server.DefaultConnectionInfo)
	message := coder.NewJsonCoder(socket)
	test.Run("TestServerNoParams", func(t *testing.T) {
		header := &coder.MessageHeader{
			ServiceDotMethod: "NoParams",
			SequenceNumber:           uint64(1),
		}
		_ = message.EncodeMessageHeaderAndBody(header, fmt.Sprintf("geerpc req %d", header.SequenceNumber))
		_ = message.DecodeMessageHeader(header)
		var reply string
		_ = message.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	})

	test.Run("TestServerEmpty", func(t *testing.T) {
		header := &coder.MessageHeader{}
		_ = message.EncodeMessageHeaderAndBody(header, fmt.Sprintf("geerpc req %d", header.SequenceNumber))
		_ = message.DecodeMessageHeader(header)
		var reply string
		_ = message.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	})

	test.Run("TestServerNormal", func(t *testing.T) {
		header := &coder.MessageHeader{
			ServiceDotMethod: "Test.Echo",
			SequenceNumber:   uint64(666),
		}
		_ = message.EncodeMessageHeaderAndBody(header, fmt.Sprintf("geerpc req %d", header.SequenceNumber))
		_ = message.DecodeMessageHeader(header)
		var reply string
		_ = message.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	})

}

