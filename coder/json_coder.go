package coder

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

type JsonCoder struct {
	connection io.ReadWriteCloser
	buffer     *bufio.Writer
	decoder    *json.Decoder
	encoder    *json.Encoder
}

func (coder *JsonCoder) ReadHeader(header *Header) error {
	log.Println("JsonCoder decoding message header...")
	return coder.decoder.Decode(header)
}

func (coder *JsonCoder) ReadBody(body interface{}) error {
	log.Println("JsonCoder decoding message body...")
	return coder.decoder.Decode(body)
}

func (coder *JsonCoder) Write(header *Header, body interface{}) (error error) {
	log.Println("JsonCoder encoding message header...")
	error = coder.encoder.Encode(header)
	if error != nil {
		log.Println("Error in encoding header:", error)
		return error
	}
	error = coder.encoder.Encode(body)
	if error != nil {
		log.Println("Error in encoding header:", error)
		return error
	}
	defer func() {
		_ = coder.buffer.Flush()
		runtimeError := recover()
		if runtimeError != nil {
			log.Println("Error in encoding process:", runtimeError)
		}
		if error != nil {
			_ = coder.Close()
		}
	}()
	return error
}

func (coder *JsonCoder) Close() error {
	log.Println("JsonCoder closing connection...")
	return coder.connection.Close()
}

var _ Coder = (*JsonCoder)(nil)

func NewJsonCoder(connection io.ReadWriteCloser) Coder {
	buffer := bufio.NewWriter(connection)
	return &JsonCoder{
		connection: connection,
		buffer:     buffer,
		decoder:    json.NewDecoder(connection),
		encoder:    json.NewEncoder(buffer),
	}
}
