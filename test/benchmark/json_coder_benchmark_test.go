package benchmark

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

func BenchmarkJsonCoder(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	time.Sleep(time.Second)

	//create test needed variables
	var waitGroup sync.WaitGroup
	var testServer *server.Server
	var testConnection io.ReadWriteCloser
	var testJsonCoder coder.Coder
	var testCase ArithmeticCase
	var requestHeader *coder.MessageHeader
	var requestBody *Input
	var responseHeader *coder.MessageHeader
	var responseBody *Output
	var err error

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

	//create server for testing
	serverChannel := make(chan *server.Server)
	waitGroup.Add(1)
	go createServer(":9011", serviceList, serverChannel, &waitGroup)
	testServer = <-serverChannel
	waitGroup.Wait()

	// create connection and JsonCoder for testing
	testConnection, err = net.Dial("tcp", testServer.Listener.Addr().String())
	if err != nil {
		log.Fatal("BenchmarkJsonCoder -> main error: RPC client dialing issue:", err)
	}
	_ = json.NewEncoder(testConnection).Encode(server.DefaultConnectionInfo)
	testJsonCoder = coder.NewJsonCoder(testConnection)

	b.Run("NewJsonCoder", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = coder.NewJsonCoder(testConnection)
		}
	})

	b.Run("EncodeMessageHeaderAndBody", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
			b.StopTimer()
			_ = testJsonCoder.DecodeMessageHeader(responseHeader)
			_ = testJsonCoder.DecodeMessageBody(responseBody)
			b.StartTimer()
		}
	})

	b.Run("DecodeMessageHeader", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
			b.StartTimer()
			_ = testJsonCoder.DecodeMessageHeader(responseHeader)
			b.StopTimer()
			_ = testJsonCoder.DecodeMessageBody(responseBody)
			b.StartTimer()
		}
	})

	//DecodeMessageBody() itself runs too fast, so it takes forever to finish the loop if used b.N as cap
	//Thus, use concrete cap 1000 which will still literates 1000000000 times when benchmark testing based on configuration of testing.B
	b.Run("DecodeMessageBody", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < 1000; i++ {
			b.StopTimer()
			_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
			_ = testJsonCoder.DecodeMessageHeader(responseHeader)
			b.StartTimer()
			_ = testJsonCoder.DecodeMessageBody(responseBody)
		}
	})
}
