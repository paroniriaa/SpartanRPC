package coder

import (
	"io"
	"log"
)

/*
	ServiceDotMethod  is the format for specific service and method: "Service.Method"
	SequenceNumber is the sequence number chosen by client
	Error is the error message from server's response if the rpc call failed
*/
type MessageHeader struct {
	ServiceDotMethod string
	SequenceNumber   uint64
	Error            string
}

type Coder interface {
	io.Closer
	DecodeMessageHeader(*MessageHeader) error
	DecodeMessageBody(interface{}) error
	EncodeMessageHeaderAndBody(*MessageHeader, interface{}) error
}

type CoderInitializer func(io.ReadWriteCloser) Coder

type CoderType string

const (
	Json CoderType = "application/json"
)

var CoderInitializerMap map[CoderType]CoderInitializer

func init() {
	CoderInitializerMap = make(map[CoderType]CoderInitializer)
	CoderInitializerMap[Json] = NewJsonCoder
	log.Printf("RPC coder -> init: Initialized CoderInitializerMap with field -> %+v", CoderInitializerMap)
}
