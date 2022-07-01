package coder

import (
	"io"
)

type Header struct {
	ServiceMethod  string // format "Service.Method"
	SequenceNumber uint64 // sequence number chosen by client
	Error          string //
}

type Coder interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCoderFunc func(io.ReadWriteCloser) Coder

type Type string

const (
	Json Type = "application/json"
)

var NewCoderFuncMap map[Type]NewCoderFunc

func init() {
	NewCoderFuncMap = make(map[Type]NewCoderFunc)
	NewCoderFuncMap[Json] = NewJsonCoder
}
