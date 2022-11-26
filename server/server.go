package server

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/service"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

// TODO: const
const MagicNumber = 0x3bef5c

// TODO: struct
type Server struct {
	Listener      net.Listener
	ServerAddress string
	ServiceMap    sync.Map
}

type Request struct {
	header  *coder.MessageHeader
	input   reflect.Value
	output  reflect.Value
	method  *service.Method
	service *service.Service
}

type ConnectionInfo struct {
	IDNumber          int
	CoderType         coder.CoderType
	ConnectionTimeout time.Duration
	ProcessingTimeout time.Duration
}

var DefaultConnectionInfo = &ConnectionInfo{
	IDNumber:          MagicNumber,
	CoderType:         coder.Json,
	ConnectionTimeout: time.Second * 10,
}

var invalidRequest = struct{}{}

func CreateServer(listener net.Listener) (*Server, error) {
	if listener == nil {
		return nil, errors.New("RPC server > CreateServer error: Network listener should not be nil, but received nil")
	}
	serverAddress := listener.Addr().Network() + "@" + listener.Addr().String()
	log.Printf("RPC server -> CreateServer: Created RPC server on address %s", serverAddress)
	return &Server{
		ServerAddress: serverAddress,
		Listener:      listener,
	}, nil
}

func CreateServerHTTP(listener net.Listener) (*Server, error) {
	if listener == nil {
		return nil, errors.New("RPC server > CreateServer error: Network listener should not be nil, but received nil")
	}
	//http@serverAddress will hide the transportation protocol info, use listener.Addr().Network() to fetch when needed
	serverAddress := "http" + "@" + listener.Addr().String()
	log.Printf("RPC server -> CreateServer: Created RPC server on address %s", serverAddress)
	return &Server{
		ServerAddress: serverAddress,
		Listener:      listener,
	}, nil
}

func (server *Server) ServerRegister(serviceValue interface{}) error {
	newService := service.CreateService(serviceValue)
	_, duplicate := server.ServiceMap.LoadOrStore(newService.ServiceName, newService)
	if duplicate {
		return errors.New("RPC server > ServerRegister error: Service has already been defined: " + newService.ServiceName)
	}
	log.Printf("RPC server -> ServerRegister: RPC server %p registerd service %+v and hosting on address %s", server, newService, server.ServerAddress)
	return nil
}

func (server *Server) LaunchAndAccept() {
	for {
		connection, err := server.Listener.Accept()
		if err != nil {
			log.Printf("RPC server -> LaunchAndAccept error: %s", err)
			return
		}
		go server.ServeConnection(connection)
	}
}

func (server *Server) ServeConnection(connection io.ReadWriteCloser) {
	defer func() { _ = connection.Close() }()
	var connectionInfo ConnectionInfo
	if err := json.NewDecoder(connection).Decode(&connectionInfo); err != nil {
		log.Printf("RPC server -> ServeConnection error: Fail to decode connectionInfo due to %s", err)
		return
	}
	if connectionInfo.IDNumber != MagicNumber {
		log.Printf("RPC server -> ServeConnection error: Invalid ID number %x", connectionInfo.IDNumber)
		return
	}
	coderInitializerFunction := coder.CoderInitializerMap[connectionInfo.CoderType]
	if coderInitializerFunction == nil {
		log.Printf("RPC server-> ServeConnection error: Invalid coder type %s", connectionInfo.CoderType)
		return
	}
	server.serveCoder(coderInitializerFunction(connection), &connectionInfo)
}

func (server *Server) serveCoder(message coder.Coder, connectionInfo *ConnectionInfo) {
	sending := new(sync.Mutex)
	waitGroup := new(sync.WaitGroup)
	for {
		requests, err := server.read_request(message)
		if requests == nil && err != nil {
			break
		} else if err != nil {
			requests.header.Error = err.Error()
			server.send_response(message, requests.header, invalidRequest, sending)
			continue
		}
		waitGroup.Add(1)
		go server.request_handle(message, requests, sending, waitGroup, connectionInfo.ProcessingTimeout)
	}
	waitGroup.Wait()
	_ = message.Close()
}

/*
func (server *Server) Connection_handle(listening net.Listener) {
	for {
		connection, err_msg := listening.Accept()
		if err_msg != nil {
			log.Println("Server - accept error:", err_msg)
			return
		}

		defer func() { _ = connection.Close() }()
		var connectionInfo ConnectionInfo
		err := json.NewDecoder(connection).Decode(&connectionInfo)
		switch {
		case err != nil:
			log.Println("Server - connectionInfo error: ", err)
			return
		case connectionInfo.IDNumber != MagicNumber:
			log.Printf("Server - ID number error: %x is invalid", connectionInfo.IDNumber)
			return
		case coder.CoderInitializerMap[connectionInfo.CoderType] == nil:
			log.Printf("Server - invalid coder type error: %s", connectionInfo.CoderType)
			return
		}
		coder_function_map := coder.CoderInitializerMap[connectionInfo.CoderType]
		server.server_coder(coder_function_map(connection), &connectionInfo)
	}
}

func (server *Server) server_coder(message coder.Coder, connectionInfo *ConnectionInfo) {
	sending := new(sync.Mutex)
	waitGroup := new(sync.WaitGroup)
	for {
		requests, err := server.read_request(message)
		if requests == nil && err != nil {
			break
		} else if err != nil {
			requests.header.Error = err.Error()
			server.send_response(message, requests.header, invalidRequest, sending)
			continue
		}
		waitGroup.Add(1)
		go server.request_handle(message, requests, sending, waitGroup, connectionInfo.ProcessingTimeout)
	}
	waitGroup.Wait()
	_ = message.Close()
}

*/

func (server *Server) read_header(message coder.Coder) (*coder.MessageHeader, error) {
	var h coder.MessageHeader
	err := message.DecodeMessageHeader(&h)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("RPC server -> read_header error: ", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) read_request(message coder.Coder) (*Request, error) {
	header, Error := server.read_header(message)
	if Error != nil {
		return nil, Error
	}
	requests := &Request{header: header}

	requests.service, requests.method, Error = server.search_service(header.ServiceDotMethod)
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
		log.Println("RPC server -> read_request error: ", Error)
		return requests, Error
	}
	return requests, nil

}

func (server *Server) search_service(serviceMethod string) (services *service.Service, methods *service.Method, err error) {
	splitIndex := strings.LastIndex(serviceMethod, ".")
	if splitIndex < 0 {
		err = errors.New("RPC server -> search_service error: " + serviceMethod + " ill-formed invalid.")
		return
	}
	serviceName, methodName := serviceMethod[:splitIndex], serviceMethod[splitIndex+1:]
	input, serviceStatus := server.ServiceMap.Load(serviceName)
	if !serviceStatus {
		err = errors.New("RPC server -> search_service error: " + serviceName + " serviceName didn't exist")
		return
	}
	services = input.(*service.Service)
	methods = services.ServiceMethod[methodName]
	if methods == nil {
		err = errors.New("RPC server -> search_service error: " + methodName + " methodName didn't exist")
	}
	return
}

func (server *Server) send_response(message coder.Coder, header *coder.MessageHeader, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	Error := message.EncodeMessageHeaderAndBody(header, body)
	if Error != nil {
		log.Println("RPC server -> write response error:", Error)
	}
	defer sending.Unlock()
}

func (server *Server) request_handle(message coder.Coder, request *Request, sending *sync.Mutex, waitGroup *sync.WaitGroup, timeoutLimit time.Duration) {
	serviceCallTimeoutChannel := make(chan struct{})
	responseSendTimeoutChannel := make(chan struct{})
	go func() {
		//request.service.ServiceName, request.method.MethodName
		log.Printf("RPC server -> requestHandle: RPC server %s is handling and redircting RPC call %d to service %s with inputs -> %v", server.ServerAddress, request.header.SequenceNumber, request.header.ServiceDotMethod, request.input)
		Error := request.service.Call(request.method, request.input, request.output)
		serviceCallTimeoutChannel <- struct{}{}
		if Error != nil {
			request.header.Error = Error.Error()
			server.send_response(message, request.header, invalidRequest, sending)
			responseSendTimeoutChannel <- struct{}{}
			return
		}
		server.send_response(message, request.header, request.output.Interface(), sending)
		responseSendTimeoutChannel <- struct{}{}
	}()
	if timeoutLimit == 0 {
		<-serviceCallTimeoutChannel
		<-responseSendTimeoutChannel
		return
	}
	//if time.After() receive message first (before serviceCallTimeoutChannel receive message), it indicates that RPC request handling hsa timeout
	//which means both serviceCallTimeoutChannel and responseSendTimeoutChannel will be blocked, so we need to send response right after.
	select {
	case <-time.After(timeoutLimit):
		request.header.Error = fmt.Sprintf("RPC server -> request_handle error: expect to handle RPC request within %s, but failed and timeout", timeoutLimit)
		server.send_response(message, request.header, invalidRequest, sending)
	case <-serviceCallTimeoutChannel:
		<-responseSendTimeoutChannel
	}
	defer waitGroup.Done()
}

const (
	ConnectedMessage = "200 Connected to Spartan RPC"
	DefaultRPCPath   = "/_srpc_"
	DefaultDebugPath = "/debug/srpc"
)

func (server *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "CONNECT" {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(writer, "405 must CONNECT\n")
		return
	}
	connection, _, err := writer.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("RPC server -> ServeHTTP hijacking ", request.RemoteAddr, ": ", err.Error())
		return
	}
	_, _ = io.WriteString(connection, "HTTP/1.0 "+ConnectedMessage+"\n\n")
	server.ServeConnection(connection)
}

// LaunchAndServe is a convenient approach for server to register an individual HTTP handlers and have it serve the specified path(s)
func (server *Server) LaunchAndServe() {
	log.Println("RPC server -> LaunchAndServe: RPC server initializing an HTTP multiplexer (handler) for debug...")
	serverMultiplexer := http.NewServeMux()
	serverMultiplexer.HandleFunc(DefaultRPCPath, server.ServeHTTP)
	serverMultiplexer.HandleFunc(DefaultDebugPath, HTTPDebug{server}.ServeHTTP)
	log.Printf("RPC server -> LaunchAndServe: RPC server finished initializing the HTTP multiplexer (handler) for debug, and it is serving on URL path: %s", "http://"+server.Listener.Addr().String()+DefaultDebugPath)
	_ = http.Serve(server.Listener, serverMultiplexer)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for the RPC server to register or send heartbeat to the RPC registry
func (server *Server) Heartbeat(registryURL string, heartRate time.Duration) {
	if heartRate == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registryURL
		heartRate = registry.DefaultTimeout - time.Duration(1)*time.Minute
	}
	log.Printf("RPC server -> Heartbeat: RPC server %s launched auto heartbeat sending service, and its heartbeat will be send to the RPC registry %s per %s...", server.ServerAddress, registryURL, heartRate)
	var err error
	err = server.sendHeartbeat(registryURL)
	go func() {
		heartRateTicker := time.NewTicker(heartRate)
		for err == nil {
			<-heartRateTicker.C
			err = server.sendHeartbeat(registryURL)
		}
	}()
}

func (server *Server) sendHeartbeat(registryURL string) error {
	log.Printf("RPC Server -> sendHeartbeat: RPC server %s sends heart beat to RPC registry %s...", server.ServerAddress, registryURL)
	httpClient := &http.Client{}
	httpRequest, _ := http.NewRequest("POST", registryURL, nil)
	httpRequest.Header.Set("POST-SpartanRPC-AliveServer", server.ServerAddress)
	if _, err := httpClient.Do(httpRequest); err != nil {
		log.Printf("RPC Server -> sendHeartbeat error: RPC server %s heart beat error %s", server.ServerAddress, err)
		return err
	}
	return nil
}
