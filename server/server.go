package server

import (
	"encoding/json"
	"io"
	"log"
	"net"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CoderType   coder.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   coder.JsonType,
}



// Server represents an RPC Server.
type Server struct{}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

// ServeConn runs the server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
func (server *Server) Connection_Handle(connection io.ReadWriteCloser) {
	defer func() { _ = connection.Close() }()
	var opt Option
	if err := json.NewDecoder(connection).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := coder.NewCodecFuncMap[opt.CoderType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CoderType)
		return
	}
	server.serverCoder(f(connection))
}





// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(listening net.Listener) {
	for {
		connection, error := listening.Accept()
		if connection != nil {
			log.Println("rpc server: accept error:", error)
			return
		}
		go server.Connection_Handle(conn)
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }