package benchmark

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func BenchmarkServer(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	time.Sleep(time.Second)

	//create test needed variables
	var waitGroup sync.WaitGroup
	var testRegistry *registry.Registry
	var testServer *server.Server
	var testServerHTTP *server.Server
	var testConnection io.ReadWriteCloser
	var testConnectionHTTP io.ReadWriteCloser
	var testJsonCoder coder.Coder
	var testJsonCoderHTTP coder.Coder
	var testListener net.Listener
	var testListenerHTTP net.Listener
	var testServerA *server.Server
	var testServerC *server.Server
	var testCase ArithmeticCase
	var requestHeader *coder.MessageHeader
	var requestBody *Input
	var responseHeader *coder.MessageHeader
	var responseBody *Output

	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	serviceList := []any{
		&arithmetic,
	}

	//create arithmetic test case
	testCase = ArithmeticCase{
		"Arithmetic.Addition",
		"+",
		&Input{1, 1},
		&Output{},
		2,
	}

	//initialize the header and body for both request and response based on arithmetic test case
	requestHeader = &coder.MessageHeader{
		ServiceDotMethod: testCase.ServiceDotMethod,
		SequenceNumber:   uint64(666),
	}
	requestBody = testCase.Input
	responseHeader = &coder.MessageHeader{}
	responseBody = testCase.Output

	//create registry for server usage testing
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":8041", registryChannel, &waitGroup)
	testRegistry = <-registryChannel
	waitGroup.Wait()

	//create 2 servers A and C(HTTP)
	serverChannelA := make(chan *server.Server)
	serverChannelC := make(chan *server.Server)
	waitGroup.Add(2)
	go createServer(":9041", serviceList, serverChannelA, &waitGroup)
	testServerA = <-serverChannelA
	go createServerHTTP(":9042", serviceList, serverChannelC, &waitGroup)
	testServerC = <-serverChannelC
	waitGroup.Wait()

	testListener, _ = net.Listen("tcp", ":9043")
	testServer, _ = server.CreateServer(testListener)
	testListenerHTTP, _ = net.Listen("tcp", ":9044")
	testServerHTTP, _ = server.CreateServerHTTP(testListenerHTTP)

	b.Run("CreateServer", func(b *testing.B) {
		//testListener, _ = net.Listen("tcp", ":0")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = server.CreateServer(testListener)
		}
	})

	b.Run("CreateServer.HTTP", func(b *testing.B) {
		//testListenerHTTP, _ = net.Listen("tcp", ":0")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = server.CreateServerHTTP(testListenerHTTP)
		}
	})

	b.Run("ServiceRegister", func(b *testing.B) {
		//testListener, _ = net.Listen("tcp", ":0")
		//testServer, _ = server.CreateServer(testListener)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, service := range serviceList {
				_ = testServer.ServiceRegister(service)
			}
		}
	})

	b.Run("ServiceRegister.HTTP", func(b *testing.B) {
		//testListenerHTTP, _ = net.Listen("tcp", ":0")
		//testServerHTTP, _ = server.CreateServerHTTP(testListenerHTTP)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, service := range serviceList {
				_ = testServerHTTP.ServiceRegister(service)
			}
		}
	})

	//RPC server's LaunchAndAccept() is blocking, no need to test
	//b.Run("LaunchAndAccept", func(b *testing.B) {})

	//HTTP-based ROC server's LaunchAndServe() is blocking, no need to test
	//b.Run("LaunchAndServe", func(b *testing.B) {})

	//ServeConnection:
	//LaunchAndAccept() -> request comes in -> ServeConnection() ->
	//serveCoder() -> read_request() -> read_header() -> search_service() -> request_handle() -> send_response()
	b.Run("ServeConnection", func(b *testing.B) {
		testConnection, _ = net.Dial("tcp", testServerA.Listener.Addr().String())
		_ = json.NewEncoder(testConnection).Encode(server.DefaultConnectionInfo)
		testJsonCoder = coder.NewJsonCoder(testConnection)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
			b.StopTimer()
			_ = testJsonCoder.DecodeMessageHeader(responseHeader)
			_ = testJsonCoder.DecodeMessageBody(responseBody)
			b.StartTimer()
		}
	})

	//ServeConnection.HTTP:
	//LaunchAndServe() -> request comes in -> ServeHTTP() -> ServeConnection() ->
	//serveCoder() -> read_request() -> read_header() -> search_service() -> request_handle() -> send_response()
	b.Run("ServeConnection.HTTP", func(b *testing.B) {
		testConnectionHTTP, _ = net.Dial("tcp", testServerC.Listener.Addr().String())
		_, _ = io.WriteString(testConnectionHTTP, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", server.DefaultServerPath))
		// Require successful HTTP response before switching to RPC protocol.
		response, err := http.ReadResponse(bufio.NewReader(testConnectionHTTP), &http.Request{Method: "CONNECT"})
		if err != nil || response.Status != server.ConnectedToServerMessage {
			b.Error(response.Status, err)
		}
		_ = json.NewEncoder(testConnectionHTTP).Encode(server.DefaultConnectionInfo)
		testJsonCoderHTTP = coder.NewJsonCoder(testConnectionHTTP)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testJsonCoderHTTP.EncodeMessageHeaderAndBody(requestHeader, requestBody)
			b.StopTimer()
			_ = testJsonCoderHTTP.DecodeMessageHeader(responseHeader)
			_ = testJsonCoderHTTP.DecodeMessageBody(responseBody)
			b.StartTimer()
		}
	})

	//WriteString("CONNECT") -> ServeHTTP() -> "CONNECT" case handling
	b.Run("ServeHTTP.POST", func(b *testing.B) {
		testConnectionHTTP, _ = net.Dial("tcp", testServerC.Listener.Addr().String())
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = io.WriteString(testConnectionHTTP, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", server.DefaultServerPath))
		}
	})

	b.Run("Heartbeat", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testServerA.Heartbeat(testRegistry.RegistryURL, 0)
		}
	})

	b.Run("Heartbeat.HTTP", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testServerC.Heartbeat(testRegistry.RegistryURL, 0)
		}
	})

}
