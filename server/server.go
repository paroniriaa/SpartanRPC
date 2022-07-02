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
type Server struct{}

func New_server() *Server {
	return &Server{}
}

type request struct {
	header       *coder.Header
	argv, replyv reflect.Value
}

type Option struct {
	IDNumber  int
	CoderType coder.Type
}

//TODO: variable
var DefaultOption = &Option{
	IDNumber:  MagicNumber,
	CoderType: coder.Json,
}

var default_server = New_server()

var invalidRequest = struct{}{}

//TODO: function
func (server *Server) Connection_handle(listening net.Listener) {
	for {
		connection, err_msg := listening.Accept()
		if err_msg != nil {
			log.Println("RPC server - accept error:", err_msg)
			return
		}

		defer func() { _ = connection.Close() }()
		var option Option
		errors := json.NewDecoder(connection).Decode(&option)
		switch {
		case errors != nil:
			log.Println("RPC server - option error: ", errors)
			return
		case option.IDNumber != MagicNumber:
			log.Printf("RPC server - ID number error: %x is invalid", option.IDNumber)
			return
		case coder.NewCoderFuncMap[option.CoderType] == nil:
			log.Printf("RPC server - coderType error:  %s coder type is invalid invalid", option.CoderType)
			return
		}
		coder_function_map := coder.NewCoderFuncMap[option.CoderType]
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
	errors := message.ReadHeader(&h)
	if errors != nil {
		if errors != io.EOF && errors != io.ErrUnexpectedEOF {
			log.Println("RPC server - readHeader error:", errors)
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
	requests.argv = reflect.New(reflect.TypeOf(""))
	err := message.ReadBody(requests.argv.Interface())
	if err != nil {
		log.Println("RPC server - readArgv err:", err)
	}
	return requests, nil
}

func (server *Server) send_response(message coder.Coder, header *coder.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	errors := message.Write(header, body)
	if errors != nil {
		log.Println("RPC server - writeResponse error:", errors)
	}
	defer sending.Unlock()
}

func (server *Server) request_handle(message coder.Coder, request *request, sending *sync.Mutex, waitGroup *sync.WaitGroup) {
	log.Println(request.header, request.argv.Elem())
	request.replyv = reflect.ValueOf(fmt.Sprintf("RPC response %d", request.header.SequenceNumber))
	server.send_response(message, request.header, request.replyv.Interface(), sending)
	defer waitGroup.Done()
}

func Connection_handle(lis net.Listener) { default_server.Connection_handle(lis) }
