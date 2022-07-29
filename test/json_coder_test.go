package test

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"log"
	"net"
	"testing"
	"time"
)

func StartServer(address chan string) {
	var test Test
	err := server.ServerRegister(&test)
	if err != nil {
		log.Fatal("Server register error:", err)
	}
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

	test.Run("EncodeMessageHeaderAndBody", func(t *testing.T) {
		err = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		_ = testJsonCoder.DecodeMessageHeader(responseHeader)
		_ = testJsonCoder.DecodeMessageBody(&responseBody)
		if err != nil {
			test.Errorf("EncodeMessageHeaderAndBody Error: %s", err)
		}
	})

	test.Run("DecodeMessageHeader", func(t *testing.T) {
		_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		err = testJsonCoder.DecodeMessageHeader(responseHeader)
		_ = testJsonCoder.DecodeMessageBody(&responseBody)
		//log.Printf("responseHeader: %+v", responseHeader)
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
		_ = testJsonCoder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
		_ = testJsonCoder.DecodeMessageHeader(responseHeader)
		err = testJsonCoder.DecodeMessageBody(&responseBody)
		//log.Printf("responseBody: %v", responseBody)
		if err != nil {
			test.Errorf("DecodeMessageBody Error: %s", err)
		}
		if responseBody != requestBody {
			test.Errorf("DecodeMessageBody Error: responseBody expected to be %s, but got %s", requestBody, responseBody)
		}
	})

}
