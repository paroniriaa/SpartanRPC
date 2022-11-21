package test

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"log"
	"net"
	"sync"
	"testing"
)

func TestJsonCoder(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	var waitGroup sync.WaitGroup

	// create service type (test is enough)
	var test Test
	services := []any{
		&test,
	}

	serverAddressChannelA := make(chan string)
	waitGroup.Add(1)
	go createServer(":0", services, serverAddressChannelA, &waitGroup)
	serverAddressA := <-serverAddressChannelA
	log.Printf("Load balancer test -> main: Server A (not registered) address fetched: %s", serverAddressA)
	waitGroup.Wait()

	connection, _ := net.Dial("tcp", serverAddressA)
	_ = json.NewEncoder(connection).Encode(server.DefaultConnectionInfo)
	testJsonCoder := coder.NewJsonCoder(connection)
	requestHeader := &coder.MessageHeader{
		ServiceDotMethod: "Test.Echo",
		SequenceNumber:   uint64(666),
	}
	requestBody := "Hello there! "
	//log.Printf("requestHeader.ServiceDotMethod: %+v", requestHeader.ServiceDotMethod)
	//log.Printf("requestHeader.SequenceNumber: %+v", requestHeader.SequenceNumber)
	//log.Printf("requestBody: %+v", requestBody)

	var err error
	var responseBody string
	responseHeader := &coder.MessageHeader{}

	t.Run("EncodeMessageHeaderAndBody", func(t *testing.T) {
		err = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		_ = testJsonCoder.DecodeMessageHeader(responseHeader)
		_ = testJsonCoder.DecodeMessageBody(&responseBody)
		if err != nil {
			t.Errorf("EncodeMessageHeaderAndBody Error: %s", err)
		}
	})

	t.Run("DecodeMessageHeader", func(t *testing.T) {
		_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		err = testJsonCoder.DecodeMessageHeader(responseHeader)
		_ = testJsonCoder.DecodeMessageBody(&responseBody)
		//log.Printf("responseHeader: %+v", responseHeader)
		if err != nil {
			t.Errorf("DecodeMessageHeader Error: %s", err)
		}
		if responseHeader.ServiceDotMethod != requestHeader.ServiceDotMethod {
			t.Errorf("DecodeMessageHeader Error: responseHeader.ServiceDotMethod expected to be %s, but got %s", requestHeader.ServiceDotMethod, responseHeader.ServiceDotMethod)
		}
		if responseHeader.SequenceNumber != requestHeader.SequenceNumber {
			t.Errorf("DecodeMessageHeader Error: responseHeader.SequenceNumber expected to be %d, but got %d", requestHeader.SequenceNumber, responseHeader.SequenceNumber)
		}
	})

	t.Run("DecodeMessageBody", func(t *testing.T) {
		_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		_ = testJsonCoder.DecodeMessageHeader(responseHeader)
		err = testJsonCoder.DecodeMessageBody(&responseBody)
		//log.Printf("responseBody: %v", responseBody)
		if err != nil {
			t.Errorf("DecodeMessageBody Error: %s", err)
		}
		if responseBody != requestBody {
			t.Errorf("DecodeMessageBody Error: responseBody expected to be %s, but got %s", requestBody, responseBody)
		}
	})

}
