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
	//log.Printf("RPC server -> CreateServer: Creating RPC server on address %s...", serverAddress)
	server := &Server{
		ServerAddress: serverAddress,
		Listener:      listener,
	}
	log.Printf("RPC server -> CreateServer: Created RPC server %s with field -> %+v", server.ServerAddress, server)
	return server, nil
}

func CreateServerHTTP(listener net.Listener) (*Server, error) {
	if listener == nil {
		return nil, errors.New("RPC server > CreateServerHTTP error: Network listener should not be nil, but received nil")
	}
	//http@serverAddress will hide the transportation protocol info, use listener.Addr().Network() to fetch the transportation protocol info when needed
	serverAddress := "http" + "@" + listener.Addr().String()
	//log.Printf("RPC server -> CreateServer: Creating HTTP RPC server on address %s...", serverAddress)
	serverHTTP := &Server{
		ServerAddress: serverAddress,
		Listener:      listener,
	}
	log.Printf("RPC server -> CreateServerHTTP: Created HTTP RPC server %s with field -> %+v", serverHTTP.ServerAddress, serverHTTP)
	return serverHTTP, nil
}

func (server *Server) ServiceRegister(serviceValue interface{}) error {
	newService := service.CreateService(serviceValue)
	_, duplicate := server.ServiceMap.LoadOrStore(newService.ServiceName, newService)
	if duplicate {
		return errors.New("RPC server > ServiceRegister error: Service has already been defined: " + newService.ServiceName)
	}
	log.Printf("RPC server -> ServiceRegister: RPC server %s registerd service %+v", server.ServerAddress, newService)
	return nil
}

func (server *Server) LaunchAndAccept() {
	log.Printf("RPC server -> LaunchAndAccept: RPC server %s launched and listening for RPC client's connection...", server.ServerAddress)
	for {
		connection, err := server.Listener.Accept()
		if err != nil {
			log.Printf("RPC server -> LaunchAndAccept error: accpet error: %s", err)
			return
		}
		log.Printf("RPC server -> LaunchAndAccept: RPC server %s accepted RPC client's connection and initialized process ServeConnection()...", server.ServerAddress)
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
	log.Printf("RPC server -> ServeConnection: RPC server %s finsihed connection serving and initialized process serveCoder()...", server.ServerAddress)
	server.serveCoder(coderInitializerFunction(connection), &connectionInfo)
}

func (server *Server) serveCoder(message coder.Coder, connectionInfo *ConnectionInfo) {
	sendMutex := new(sync.Mutex)
	waitGroup := new(sync.WaitGroup)
	log.Printf("RPC server -> serveCoder: RPC server %s initialized process read_request()...", server.ServerAddress)
	for {
		requests, err := server.read_request(message)
		if requests == nil && err != nil {
			log.Printf("RPC server -> serveCoder error: request is nil due to error: %s ", err.Error())
			break
		} else if err != nil {
			requests.header.Error = err.Error()
			log.Printf("RPC server -> serveCoder error: invalid request due to error: %s ", err.Error())
			server.send_response(message, requests.header, invalidRequest, sendMutex)
			continue
		}
		waitGroup.Add(1)
		log.Printf("RPC server -> serveCoder: RPC server %s finsihed reading request and initializing sub Go routine for process request_handle()...", server.ServerAddress)
		go server.request_handle(message, requests, sendMutex, waitGroup, connectionInfo.ProcessingTimeout)
	}
	waitGroup.Wait()
	_ = message.Close()
}

func (server *Server) read_request(message coder.Coder) (*Request, error) {
	log.Printf("RPC server -> read_request: RPC server %s initialized process read_header()...", server.ServerAddress)
	header, err := server.read_header(message)
	if err != nil {
		return nil, err
	}
	request := &Request{header: header}
	log.Printf("RPC server -> read_request: RPC server %s finished reading the request and initialized process search_service()...", server.ServerAddress)
	request.service, request.method, err = server.search_service(header.ServiceDotMethod)
	if err != nil {
		log.Printf("RPC server -> read_request error: search_service error: %s", err.Error())
		return request, err
	}
	log.Printf("RPC server -> read_request: RPC server %s finished seaching the service and initialized process CreateInput() for input and output...", server.ServerAddress)
	request.input = request.method.CreateInput()
	request.output = request.method.CreateOutput()

	input := request.input.Interface()
	if request.input.Type().Kind() != reflect.Pointer {
		input = request.input.Addr().Interface()
	}
	log.Printf("RPC server -> read_request: RPC server %s finished creating input and output type and initialized process DecodeMessageBody() for input...", server.ServerAddress)
	err = message.DecodeMessageBody(input)
	if err != nil {
		log.Printf("RPC server -> read_request error: DecodeMessageBody error: %s", err.Error())
		return request, err
	}
	log.Printf("RPC server -> read_request: RPC server %s finished decoding the message body and constructed request with field -> %+v", server.ServerAddress, request)
	return request, nil

}

func (server *Server) read_header(message coder.Coder) (*coder.MessageHeader, error) {
	var header coder.MessageHeader
	err := message.DecodeMessageHeader(&header)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Printf("RPC server -> read_header error: DecodeMessageHeader error: %s", err.Error())
		}
		return nil, err
	}
	log.Printf("RPC server -> read_header: RPC server %s finished decoding the message header and constructed header with field -> %+v", server.ServerAddress, header)
	return &header, nil
}

func (server *Server) search_service(serviceDotMethod string) (matchedService *service.Service, matchedMethod *service.Method, err error) {
	splitIndex := strings.LastIndex(serviceDotMethod, ".")
	if splitIndex < 0 {
		err = errors.New("RPC server -> search_service error:  serviceDotMethod " + serviceDotMethod + " is invalid.")
		return nil, nil, err
	}
	serviceName, methodName := serviceDotMethod[:splitIndex], serviceDotMethod[splitIndex+1:]
	input, serviceStatus := server.ServiceMap.Load(serviceName)
	if !serviceStatus {
		err = errors.New("RPC server -> search_service error: serviceName " + serviceName + " does not exist")
		return nil, nil, err
	}
	matchedService = input.(*service.Service)
	matchedMethod = matchedService.ServiceMethod[methodName]
	if matchedMethod == nil {
		err = errors.New("RPC server -> search_service error: methodName " + methodName + " does not exist")
	}
	log.Printf("RPC server -> search_service: RPC server %s finished mathcing the service %+v and method %+v", server.ServerAddress, matchedService, matchedMethod)
	return matchedService, matchedMethod, nil
}

func (server *Server) send_response(message coder.Coder, header *coder.MessageHeader, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	log.Printf("RPC server -> send_response: RPC server %s sending response and initialized EncodeMessageHeaderAndBody() for encoding message header and body...", server.ServerAddress)
	err := message.EncodeMessageHeaderAndBody(header, body)
	if err != nil {
		log.Println("RPC server -> write response error:", err)
	}
	defer sending.Unlock()
}

func (server *Server) request_handle(message coder.Coder, request *Request, sendMutex *sync.Mutex, waitGroup *sync.WaitGroup, timeoutLimit time.Duration) {
	serviceCallTimeoutChannel := make(chan struct{})
	responseSendTimeoutChannel := make(chan struct{})
	go func() {
		//request.service.ServiceName, request.method.MethodName
		log.Printf("RPC server -> request_handle: RPC server %s is handling and redircting RPC call %d to service %s with inputs -> %+v", server.ServerAddress, request.header.SequenceNumber, request.header.ServiceDotMethod, request.input)
		err := request.service.Call(request.method, request.input, request.output)
		serviceCallTimeoutChannel <- struct{}{}
		if err != nil {
			request.header.Error = err.Error()
			log.Printf("RPC server -> request_handle error: fail to handle the request due to error: %s", err.Error())
			server.send_response(message, request.header, invalidRequest, sendMutex)
			responseSendTimeoutChannel <- struct{}{}
			return
		}
		log.Printf("RPC server -> request_handle: RPC server %s successfully handled RPC call %d, initializing process send_response()...", server.ServerAddress, request.header.SequenceNumber)
		server.send_response(message, request.header, request.output.Interface(), sendMutex)
		responseSendTimeoutChannel <- struct{}{}
	}()
	if timeoutLimit == 0 {
		<-serviceCallTimeoutChannel
		<-responseSendTimeoutChannel
		return
	}
	//if time.After() receive message first (before serviceCallTimeoutChannel receive message), it indicates that RPC request handling has timeout
	//which means both serviceCallTimeoutChannel and responseSendTimeoutChannel will be blocked, so we need to send response right after.
	select {
	case <-time.After(timeoutLimit):
		log.Printf("RPC server -> request_handle error : RPC server %s expect to handle RPC call %d within %s, but failed and timeout, sending ", server.ServerAddress, request.header.SequenceNumber, timeoutLimit)
		request.header.Error = fmt.Sprintf("RPC server -> request_handle error: RPC server %s expect to handle RPC call with service.method %s and input %+v within processing time limit %s, but failed and timeout", server.ServerAddress, request.service.ServiceName+request.method.MethodName, request.input, timeoutLimit)
		server.send_response(message, request.header, invalidRequest, sendMutex)
	case <-serviceCallTimeoutChannel:
		<-responseSendTimeoutChannel
	}
	defer waitGroup.Done()
}

const (
	ConnectedToServerMessage = "200 Connected to Spartan RPC"
	DefaultServerPath        = "/_srpc_/server/connect"
	DefaultServerInfoPath    = "/_srpc_/server/info"
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
	_, _ = io.WriteString(connection, "HTTP/1.0 "+ConnectedToServerMessage+"\n\n")
	log.Printf("RPC server -> ServeHTTP: RPC server %s responsed to HTTP CONNECT request from RPC client and serving its connection...", server.ServerAddress)
	server.ServeConnection(connection)
}

// LaunchAndServe is a convenient approach for server to register an individual HTTP handlers and have it serve the specified path(s)
func (server *Server) LaunchAndServe() {
	//log.Printf("RPC server -> LaunchAndServe: RPC server %s initializing an HTTP multiplexer (handler) for HTTP message receiving/sending...", server.ServerAddress)
	serverMultiplexer := http.NewServeMux()
	var serverConnectURL string
	var serverInfoURL string
	serverListenerAddress := server.Listener.Addr().String()
	if serverListenerAddress[:4] == "[::]" {
		//serverListenerAddress -> "[::]:1234" -> port extraction needed
		serverConnectURL = "http://localhost" + serverListenerAddress[4:] + DefaultServerPath
		serverInfoURL = "http://localhost" + serverListenerAddress[4:] + DefaultServerInfoPath
	} else {
		//serverListenerAddress -> "127.0.0.1:1234", port extraction not needed
		serverConnectURL = "http://" + serverListenerAddress + DefaultServerPath
		serverInfoURL = "http://" + serverListenerAddress + DefaultServerInfoPath

	}
	serverInfoHTTP := &ServerInfoHTTP{
		Server:           server,
		ServerConnectURL: serverConnectURL,
		ServerInfoURL:    serverInfoURL,
	}
	serverMultiplexer.HandleFunc(DefaultServerPath, server.ServeHTTP)
	serverMultiplexer.HandleFunc(DefaultServerInfoPath, serverInfoHTTP.ServeHTTP)
	log.Printf("RPC server -> LaunchAndServe: RPC server %s finished initializing the HTTP multiplexer (handler), and it is serving on URL paths: sRPC server connect endpoint -> %s, sRPC server info endpoint -> %s", server.ServerAddress, serverInfoHTTP.ServerConnectURL, serverInfoHTTP.ServerInfoURL)
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
	log.Printf("RPC Server -> sendHeartbeat: RPC server %s sends its heart beat to RPC registry %s...", server.ServerAddress, registryURL)
	httpClient := &http.Client{}
	httpRequest, _ := http.NewRequest("POST", registryURL, nil)
	httpRequest.Header.Set("POST-SpartanRPC-AliveServer", server.ServerAddress)
	if _, err := httpClient.Do(httpRequest); err != nil {
		log.Printf("RPC Server -> sendHeartbeat error: RPC server %s heart beat error %s", server.ServerAddress, err)
		return err
	}
	return nil
}
