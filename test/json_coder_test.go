package test

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

func StartServer(address chan string) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("Network error:", err)
	}
	log.Println("Start RPC server on port:", listener.Addr())
	address <- listener.Addr().String()
	server.AcceptConnection(listener)
}

func TestJsonCoder(test *testing.T) {
	test.Helper()
	address := make(chan string)
	go StartServer(address)
	connection, _ := net.Dial("tcp", <-address)
	//defer func() { _ = connection.Close() }()
	time.Sleep(time.Second)
	_ = json.NewEncoder(connection).Encode(server.DefaultConnectionInfo)
	TestJsonCoder := coder.NewJsonCoder(connection)
	requestHeader := &coder.MessageHeader{
		ServiceDotMethod: `json:"Test.Echo"`,
		SequenceNumber:   uint64(0),
	}
	requestBody := "Hello there! "
	test.Run("EncodeMessageHeaderAndBody", func(t *testing.T) {
		var err error

		err = TestJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		if err != nil {
			test.Errorf("EncodeMessageHeaderAndBody Error: %s", err)
		}
	})

	test.Run("DecodeMessageHeader", func(t *testing.T) {
		var err error
		responseHeader := &coder.MessageHeader{}
		err = TestJsonCoder.DecodeMessageHeader(responseHeader)
		if err != nil {
			test.Errorf("DecodeMessageHeader Error: %s", err)
		}
		if responseHeader.ServiceDotMethod != requestHeader.ServiceDotMethod {
			test.Errorf("DecodeMessageHeader Error: responseHeader.ServiceDotMethod expected to be %s, but got %s", requestHeader.ServiceDotMethod, responseHeader.ServiceDotMethod)
		}
		if responseHeader.SequenceNumber != requestHeader.SequenceNumber {
			test.Errorf("DecodeMessageHeader Error: responseHeader.SequenceNumber expected to be %d, but got %d", requestHeader.SequenceNumber, responseHeader.SequenceNumber)
		}
	})

	test.Run("DecodeMessageBody", func(t *testing.T) {


		var err error
		var responseBody string
		err = TestJsonCoder.DecodeMessageBody(&responseBody)
		if err != nil {
			test.Errorf("DecodeMessageBody Error: %s", err)
		}
		log.Println(responseBody)
		if responseBody != requestBody {
			test.Errorf("DecodeMessageBody Error: responseBody expected to be %s, but got %s", responseBody, responseBody)
		}
	})

}



func TestJsonCoder2(test *testing.T) {
	test.Helper()
	log.SetFlags(0)
	addr := make(chan string)
	go StartServer(addr)

	// in fact, following code is like a simple geerpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	// send options
	_ = json.NewEncoder(conn).Encode(server.DefaultConnectionInfo)
	cc := coder.NewJsonCoder(conn)
	for i := 0; i < 5; i++ {
		h := &coder.MessageHeader{
			ServiceDotMethod: "Foo.Sum",
			SequenceNumber:           uint64(i),
		}
		_ = cc.EncodeMessageHeaderAndBody(h, fmt.Sprintf("geerpc req %d", h.SequenceNumber))
		_ = cc.DecodeMessageHeader(h)
		var reply string
		_ = cc.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	}




}

