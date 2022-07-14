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

var _ Coder = (*JsonCoder)(nil)

func NewJsonCoder(connection io.ReadWriteCloser) Coder {
	buffer := bufio.NewWriter(connection)
	return &JsonCoder{
		connection: connection,
		buffer:     buffer,
		encoder:    json.NewEncoder(buffer),
		decoder:    json.NewDecoder(connection),
	}
}

func (coder *JsonCoder) DecodeMessageHeader(header *MessageHeader) error {
	return coder.decoder.Decode(header)
}

func (coder *JsonCoder) DecodeMessageBody(body interface{}) error {
	return coder.decoder.Decode(body)
}

func (coder *JsonCoder) EncodeMessageHeaderAndBody(header *MessageHeader, body interface{}) (error error) {
	if error = coder.encoder.Encode(header); error != nil {
		log.Fatal("Coder Error when encoding header:", error)
		return error
	}
	if error = coder.encoder.Encode(body); error != nil {
		log.Println("Coder Error when encoding body:", error)
		return error
	}
	defer func() {
		_ = coder.buffer.Flush()
		if error != nil {
			_ = coder.Close()
		}
	}()
	return error
}

func (coder *JsonCoder) Close() error {
	return coder.connection.Close()
}
