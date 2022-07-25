package server

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/service"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

// TODO: const
const MagicNumber = 0x3bef5c

// TODO: struct
type Server struct {
	serviceMap sync.Map
}

type Request struct {
	header  *coder.Header
	input   reflect.Value
	output  reflect.Value
	method  *service.Method
	service *service.Service
}

type ConnectionInfo struct {
	IDNumber  int
	CoderType coder.CoderType
	Timeout_connection time.Duration
	Timeout_handle  time.Duration
}

func New_server() *Server {
	return &Server{}
}

//TODO: variable
var DefaultConnectionInfo = &ConnectionInfo{
	IDNumber:  MagicNumber,
	CoderType: coder.Json,
	Timeout_connection: time.Second * 10,
}

var default_server = New_server()

var invalidRequest = struct{}{}

//TODO: function
func (server *Server) searchService(serviceMethod string) (services *service.Service, methods *service.Method, err error) {
	splitIndex := strings.LastIndex(serviceMethod, ".")
	if splitIndex < 0 {
		err = errors.New("Server - searchService error: " + serviceMethod + " ill-formed invalid.")
		return
	}
	serviceName, methodName := serviceMethod[:splitIndex], serviceMethod[splitIndex+1:]
	input, serviceStatus := server.serviceMap.Load(serviceName)
	if !serviceStatus {
		err = errors.New("Server - searchService error: " + serviceName + " serviceName didn't exist")
		return
	}
	services = input.(*service.Service)
	methods = services.ServiceMethod[methodName]
	if methods == nil {
		err = errors.New("Server - searchService error: " + methodName + " methodName didn't exist")
	}
	return
}

func (server *Server) Connection_handle(listening net.Listener) {
	for {
		connection, err_msg := listening.Accept()
		if err_msg != nil {
			log.Println("Server - accept error:", err_msg)
			return
		}

		defer func() { _ = connection.Close() }()
		var option ConnectionInfo
		errors := json.NewDecoder(connection).Decode(&option)
		switch {
		case errors != nil:
			log.Println("Server - option error: ", errors)
			return
		case option.IDNumber != MagicNumber:
			log.Printf("Server - ID number error: %x is invalid", option.IDNumber)
			return
		case coder.CoderFunctionMap[option.CoderType] == nil:
			log.Printf("Server - invalid coder type error: %s", option.CoderType)
			return
		}
		coder_function_map := coder.CoderFunctionMap[option.CoderType]
		server.server_coder(coder_function_map(connection), &option)
	}
}

func (server *Server) server_coder(message coder.Coder, option *ConnectionInfo) {
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
		go server.request_handle(message, requests, sending, waitGroup,option.Timeout_handle)
	}
	waitGroup.Wait()
	_ = message.Close()
}

func (server *Server) read_header(message coder.Coder) (*coder.Header, error) {
	var h coder.Header
	errors := message.DecodeMessageHeader(&h)
	if errors != nil {
		if errors != io.EOF && errors != io.ErrUnexpectedEOF {
			log.Println("Server - read header error:", errors)
		}
		return nil, errors
	}
	return &h, nil
}

func (server *Server) read_request(message coder.Coder) (*Request, error) {
	header, Error := server.read_header(message)
	if Error != nil {
		return nil, Error
	}
	requests := &Request{header: header}

	requests.service, requests.method, Error = server.searchService(header.ServiceDotMethod)
	if Error != nil {
		return requests, Error
	}
	requests.input = requests.method.CreateInput()
	requests.output = requests.method.CreateOutput()

	input := requests.input.Interface()
	if requests.input.Type().Kind() != reflect.Pointer {
		input = requests.input.Addr().Interface()
	}

	Error = message.DecodeMessageBody(input)
	if Error != nil {
		log.Println("Server - DecodeMessageBody error:", Error)
		return requests, Error
	}
	return requests, nil

}

func (server *Server) send_response(message coder.Coder, header *coder.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	err := message.EncodeMessageHeaderAndBody(header, body)
	if err != nil {
		log.Println("Server - write response error:", err)
	}
	defer sending.Unlock()
}

//old handleRequest
/*
func (server *Server) request_handle(message coder.Coder, request *Request, sending *sync.Mutex, waitGroup *sync.WaitGroup) {
	Error := request.service.Call(request.method, request.input, request.output)
	if Error != nil {
		request.header.Error = Error.Error()
		server.send_response(message, request.header, invalidRequest, sending)
		return
	}
	server.send_response(message, request.header, request.output.Interface(), sending)
	defer waitGroup.Done()
}
*/

func (server *Server) request_handle(message coder.Coder, request *Request, sending *sync.Mutex, waitGroup *sync.WaitGroup, timeout time.Duration) {
	defer waitGroup.Done()
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := request.service.Call(request.method, request.input, request.output)
		called <- struct{}{}
		if err != nil {
			request.header.Error = err.Error()
			server.send_response(message, request.header, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
		server.send_response(message, request.header, request.output.Interface(), sending)
		sent <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-sent
		return
	}
	select {
	case <-time.After(timeout):
		request.header.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.send_response(message, request.header, invalidRequest, sending)
	case <-called:
		<-sent
	}
}


func Connection_handle(lis net.Listener) { default_server.Connection_handle(lis) }

func (server *Server) ServerRegister(serviceValue interface{}) error {
	newService := service.CreateService(serviceValue)
	_, duplicate := server.serviceMap.LoadOrStore(newService.ServiceName, newService)
	if duplicate {
		return errors.New("Server - Connection_handle error: Service has already been defined: " + newService.ServiceName)
	}
	return nil
}

func ServerRegister(serviceValue interface{}) error {
	return default_server.ServerRegister(serviceValue)
}
