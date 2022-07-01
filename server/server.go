package server

import (
	"Distributed-RPC-Framework/coder"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
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


// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(listening net.Listener) {
	for {
		connection, error := listening.Accept()
		if connection != nil {
			log.Println("rpc server: accept error:", error)
			return
		}

		defer func() { _ = connection.Close() }()
		var opt Option
		err := json.NewDecoder(connection).Decode(&opt)
		switch {
		case err != nil:
			log.Println("rpc server: options error: ", err)
			return
		case opt.MagicNumber != MagicNumber:
			log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
			return
		case coder.NewCodecFuncMap[opt.CoderType] == nil:
			log.Printf("rpc server: invalid codec type %s", opt.CoderType)
			return
		}

		server.serverCoder(f(connection))
	}
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc coder.Codec) {
	sending := new(sync.Mutex)
	waitGroup := new(sync.WaitGroup)
	for{
		req,err := server.reqdRequest(cc)
		if req == nil && err != nil {
			break
		} else {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		waitGroup.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	waitGroup.Wait()
	_ = cc.Close()
}

// request stores all information of a call
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: now we don't know the type of request argv
	// day 1, just suppose it's string
	req.argv = reflect.New(reflect.TypeOf(""))
	err := cc.ReadBody(req.argv.Interface())
	if err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}


func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	err := cc.Write(h, body)
	if err != nil {
		log.Println("rpc server: write response error:", err)
	}
	defer sending.Unlock()
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv
	// day 1, just print argv and send a hello message
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
	defer wg.Done()
}



// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }