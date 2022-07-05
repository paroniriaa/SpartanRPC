package server

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/service"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
)

// TODO: const
const MagicNumber = 0x3bef5c

// TODO: struct
type Server struct {
	serviceMap sync.Map
}

func New_server() *Server {
	return &Server{}
}

type request struct {
	header       *coder.Header
	argv, replyv reflect.Value
	serviceType  *service.ServiceType
	service      *service.Service
}

type Option struct {
	IDNumber  int
	CoderType coder.CoderType
}

//TODO: variable
var DefaultOption = &Option{
	IDNumber:  MagicNumber,
	CoderType: coder.Json,
}

var default_server = New_server()

var invalidRequest = struct{}{}

//TODO: function
func (server *Server) searchService(serviceMethod string) (services *service.Service, methodType *service.ServiceType, Error error) {
	splitIndex := strings.LastIndex(serviceMethod, ".")
	if splitIndex < 0 {
		Error = errors.New("Server - searchService error: " + serviceMethod + " ill-formed invalid.")
		return
	}
	serviceName, methodName := serviceMethod[:splitIndex], serviceMethod[splitIndex+1:]
	input, isServiceStatus := server.serviceMap.Load(serviceName)
	if !isServiceStatus {
		Error = errors.New("Server - searchService error: " + serviceName + " serviceName didn't exist")
		return
	}
	services = input.(*service.Service)
	methodType = services.ServiceMethod[methodName]
	if methodType == nil {
		Error = errors.New("Server - searchService error: " + methodName + " methodName didn't exist")
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
		var option Option
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
			log.Println("Server - read header error:", errors)
		}
		return nil, errors
	}
	return &h, nil
}

func (server *Server) read_request(message coder.Coder) (*request, error) {
	header, Error := server.read_header(message)
	if Error != nil {
		return nil, Error
	}
	requests := &request{header: header}

	requests.service, requests.serviceType, Error = server.searchService(header.ServiceMethod)
	if Error != nil {
		return requests, Error
	}
	requests.argv = requests.serviceType.GetInput()
	requests.replyv = requests.serviceType.GetOutput()

	input := requests.argv.Interface()
	if requests.argv.Type().Kind() != reflect.Pointer {
		input = requests.argv.Addr().Interface()
	}

	Error = message.DecodeMessageBody(input)
	if Error != nil {
		log.Println("Server - read_request error:", Error)
		return requests, Error
	}
	return requests, nil

}

func (server *Server) send_response(message coder.Coder, header *coder.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	errors := message.EncodeMessageHeaderAndBody(header, body)
	if errors != nil {
		log.Println("Server - write response error:", errors)
	}
	defer sending.Unlock()
}

func (server *Server) request_handle(message coder.Coder, request *request, sending *sync.Mutex, waitGroup *sync.WaitGroup) {
	Error := request.service.Call(request.serviceType, request.argv, request.replyv)
	if Error != nil {
		request.header.Error = Error.Error()
		server.send_response(message, request.header, invalidRequest, sending)
		return
	}
	server.send_response(message, request.header, request.replyv.Interface(), sending)
	defer waitGroup.Done()
}

func Connection_handle(lis net.Listener) { default_server.Connection_handle(lis) }

func (server *Server) ServerRegistry(serviceValue interface{}) error {
	newService := service.CreateService(serviceValue)
	_, duplicate := server.serviceMap.LoadOrStore(newService.ServiceName, newService)
	if duplicate {
		return errors.New("Server - Connection_handle error: Service has already been defined: " + newService.ServiceName)
	}
	return nil
}

func ServerRegistry(serviceValue interface{}) error {
	return default_server.ServerRegistry(serviceValue)
}
