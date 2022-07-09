package coder

import (
	"io"
)

/*
	ServiceDotMethod  is the format for specific service and method: "Service.Method"
	SequenceNumber is the sequence number chosen by client
	Error is the error message from server's response if the rpc call failed
*/
type Message struct {
	ServiceDotMethod string
	SequenceNumber   uint64
	Error            string
}

type Coder interface {
	io.Closer
	/*
		Connection() io.ReadWriteCloser
		Buffer() *bufio.Writer
		Encoder() *json.Encoder
		Decoder() *json.Decoder
	*/
	DecodeMessageHeader(*Message) error
	DecodeMessageBody(interface{}) error
	EncodeMessageHeaderAndBody(*Message, interface{}) error
}

type CoderFunction func(io.ReadWriteCloser) Coder

type CoderType string

const (
	Json CoderType = "application/json"
)

var CoderFunctionMap map[CoderType]CoderFunction

func init() {
	CoderFunctionMap = make(map[CoderType]CoderFunction)
	CoderFunctionMap[Json] = NewJsonCoder
}
