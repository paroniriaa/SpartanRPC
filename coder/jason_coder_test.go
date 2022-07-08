package coder

import (
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"testing"
)

func CreateConnection() (connection net.Conn) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr := make(chan string)
	addr <- l.Addr().String()
	server.Accept(l)
	connection, _ = net.Dial("tcp", <-addr)
	return connection
}

func TestNewJsonCoder(test *testing.T) {
	test.Helper()
	connection := CreateConnection()
	_ = json.NewEncoder(connection).Encode(server.DefaultOption)
	connectionCoder := NewJsonCoder(connection)
	// send request & receive response
	for i := 0; i < 5; i++ {
		header := &Header{
			ServiceMethod:  "Test.Echo",
			SequenceNumber: uint64(i),
		}
		request := "RPC request " + strconv.Itoa(i)
		_ = connectionCoder.Write(header, request)
		_ = connectionCoder.ReadHeader(header)
		var reply string
		_ = connectionCoder.ReadBody(&reply)
		expectedReply := request
		if reply != expectedReply {
			test.Errorf("Reply expected be %s, but got %s", expectedReply, reply)
		}
	}
	defer func() { _ = connection.Close() }()
}
