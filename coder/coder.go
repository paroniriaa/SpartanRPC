package coder

import (
	"io"
)

/*
	ServiceMethod  is the format for specific service and method: "Service.Method"
	SequenceNumber is the sequence number chosen by client
	Error is the error message from server's response if the rpc call failed
*/
type Header struct {
	ServiceMethod  string
	SequenceNumber uint64
	Error          string
}

type Coder interface {
	io.Closer
	/*
		Connection() io.ReadWriteCloser
		Buffer() *bufio.Writer
		Encoder() *json.Encoder
		Decoder() *json.Decoder
	*/
	DecodeMessageHeader(*Header) error
	DecodeMessageBody(interface{}) error
	EncodeMessageHeaderAndBody(*Header, interface{}) error
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
