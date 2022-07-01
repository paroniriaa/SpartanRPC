package coder

import (
	"io"
)

type Type string

type NewCoderFunc func(io.ReadWriteCloser) Coder

type Header struct {
	ServiceMethod string // the format for specific service and method: "Service.Method"
	SeqNumber     uint64 // the sequence number chosen by client
	Error         string // error message from server's response if the rpc call failed
}

type Coder interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

const (
	Json Type = "application/json"
)

var NewCoderFuncMap map[Type]NewCoderFunc

func Init() {
	NewCoderFuncMap = make(map[Type]NewCoderFunc)
	NewCoderFuncMap[Json] = NewJsonCoder
}
