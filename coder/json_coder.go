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
	newJsonCoder := &JsonCoder{
		connection: connection,
		buffer:     buffer,
		encoder:    json.NewEncoder(buffer),
		decoder:    json.NewDecoder(connection),
	}
	log.Printf("RPC JsonCoder -> NewJsonCoder: RPC JsonCoder %p created with field -> %+v", newJsonCoder, newJsonCoder)
	return newJsonCoder
}

func (jsonCoder *JsonCoder) DecodeMessageHeader(header *MessageHeader) error {
	log.Printf("RPC JsonCoder -> DecodeMessageHeader: RPC JsonCoder %p deconding message header...", jsonCoder)
	return jsonCoder.decoder.Decode(header)
}

func (jsonCoder *JsonCoder) DecodeMessageBody(body interface{}) error {
	log.Printf("RPC JsonCoder -> DecodeMessageBody: RPC JsonCoder %p deconding message body...", jsonCoder)
	return jsonCoder.decoder.Decode(body)
}

func (jsonCoder *JsonCoder) EncodeMessageHeaderAndBody(header *MessageHeader, body interface{}) (error error) {
	log.Printf("RPC JsonCoder -> EncodeMessageHeaderAndBody: RPC JsonCoder %p encoding message header and body...", jsonCoder)
	if error = jsonCoder.encoder.Encode(header); error != nil {
		log.Println("RPC Json Coder -> EncodeMessageHeaderAndBody Error: error when encoding header:", error)
		return error
	}
	if error = jsonCoder.encoder.Encode(body); error != nil {
		log.Println("RPC Json Coder -> EncodeMessageHeaderAndBody Error: error when encoding body:", error)
		return error
	}
	defer func() {
		_ = jsonCoder.buffer.Flush()
		if error != nil {
			_ = jsonCoder.Close()
		}
	}()
	return error
}

func (jsonCoder *JsonCoder) Close() error {
	log.Printf("RPC JsonCoder -> Close: RPC JsonCoder %p is closing...", jsonCoder)
	return jsonCoder.connection.Close()
}
