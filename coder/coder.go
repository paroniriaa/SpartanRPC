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
		decoder:    json.NewDecoder(connection),
		encoder:    json.NewEncoder(buffer),
	}
}

func (coder *JsonCoder) ReadHeader(h *Header) error {
	return coder.decoder.Decode(h)
}

func (coder *JsonCoder) ReadBody(body interface{}) error {
	return coder.decoder.Decode(body)
}

func (coder *JsonCoder) Write(h *Header, body interface{}) (err error) {
	if err = coder.encoder.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return
	}
	if err = coder.encoder.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return
	}
	defer func() {
		_ = coder.buffer.Flush()
		if err != nil {
			_ = coder.Close()
		}
	}()
	return
}

func (coder *JsonCoder) Close() error {
	return coder.connection.Close()
}
