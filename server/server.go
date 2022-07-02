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

// TODO: const
const MagicNumber = 0x3bef5c

// TODO: struct
// Server represents an RPC Server.
type Server struct{}

// NewServer returns a new Server.
func New_server() *Server {
	return &Server{}
}

// request stores all information of a call
type request struct {
	header       *coder.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

type Option struct {
	IDNumber  int             // MagicNumber marks this's a geerpc request
	CoderType coder.CoderType // client may choose different Codec to encode body
}

//TODO: variable
var DefaultOption = &Option{
	IDNumber:  MagicNumber,
	CoderType: coder.Json,
}

// DefaultServer is the default instance of *Server.
var default_server = New_server()

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

//@function
// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Connection_handle(listening net.Listener) {
	for {
		connection, err_msg := listening.Accept()
		if err_msg != nil {
			log.Println("rpc server: accept error:", err_msg)
			return
		}

		defer func() { _ = connection.Close() }()
		var option Option
		errors := json.NewDecoder(connection).Decode(&option)
		switch {
		case errors != nil:
			log.Println("rpc option error: ", errors)
			return
		case option.IDNumber != MagicNumber:
			log.Printf("rpc ID number error: %x is invalid", option.IDNumber)
			return
		case coder.CoderFunctionMap[option.CoderType] == nil:
			log.Printf("rpc server: invalid codec type %s", option.CoderType)
			return
		}
		coder_function_map := coder.CoderFunctionMap[option.CoderType]
		server.server_coder(coder_function_map(connection))
	}
}

func (server *Server) server_coder(message coder.Coder) {
	sending := new(sync.Mutex)
	waitGroup := new(sync.WaitGroup)
	for {
		requests, errors := server.read_request(message)
		if requests == nil && errors != nil {
			break
		} else if errors != nil {
			requests.header.Error = errors.Error()
			server.send_response(message, requests.header, invalidRequest, sending)
			continue
		}
		waitGroup.Add(1)
		go server.request_handle(message, requests, sending, waitGroup)
	}
	waitGroup.Wait()
	_ = message.Close()
}

func (server *Server) read_header(message coder.Coder) (*coder.Header, error) {
	var h coder.Header
	errors := message.DecodeMessageHeader(&h)
	if errors != nil {
		if errors != io.EOF && errors != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", errors)
		}
		return nil, errors
	}
	return &h, nil
}

func (server *Server) read_request(message coder.Coder) (*request, error) {
	header, errors := server.read_header(message)
	if errors != nil {
		return nil, errors
	}
	requests := &request{header: header}
	// TODO: now we don't know the type of request argv
	// day 1, just suppose it's string
	requests.argv = reflect.New(reflect.TypeOf(""))
	err := message.DecodeMessageBody(requests.argv.Interface())
	if err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return requests, nil
}

func (server *Server) send_response(message coder.Coder, header *coder.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	errors := message.EncodeMessageHeaderAndBody(header, body)
	if errors != nil {
		log.Println("rpc server: write response error:", errors)
	}
	defer sending.Unlock()
}

func (server *Server) request_handle(message coder.Coder, request *request, sending *sync.Mutex, waitGroup *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv
	// day 1, just print argv and send a hello message
	log.Println(request.header, request.argv.Elem())
	request.replyv = reflect.ValueOf(fmt.Sprintf("%s", request.argv.Elem()))
	server.send_response(message, request.header, request.replyv.Interface(), sending)
	defer waitGroup.Done()
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Connection_handle(lis net.Listener) { default_server.Connection_handle(lis) }
