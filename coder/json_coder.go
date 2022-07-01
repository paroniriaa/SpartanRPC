package coder

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"io"
)

Type JsonCoder struct {
	connection io.ReadWriteCloser
	buffer  *bufio.Writer
	decoder  *json.Decoder
	encoder  *json.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(buf),
	}
}