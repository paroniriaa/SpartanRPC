package client

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"bufio"
	"context"
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

// Call represents an active RPC.
type Call struct {
	SequenceNumber   uint64      // the sequence number of the RPC call
	ServiceDotMethod string      // the service and method to call (format: service.method)
	Inputs           interface{} // the inputs to the function (*struct)
	Output           interface{} // The output from the function (*struct).
	Error            error       // After completion, the error status.
	Finish           chan *Call  // Receives *Call when the call that Go channels handle is finished
}

// Client represents an RPC Client.
type Client struct {
	coder          coder.Coder            // the message coder for message encoding and decoding
	connectionInfo *server.ConnectionInfo // the connection (between server and client ) info

	requestMutex  sync.Mutex          // protects following
	requestHeader coder.MessageHeader // the header of an RPC request

	entityMutex    sync.Mutex       // protects following
	sequenceNumber uint64           // the sequence number that counts the RPC request
	pendingCalls   map[uint64]*Call // the map that store all pending RPC calls, key is Call.SequenceNumber, value is Call
	isClosed       bool             // close by client
	isShutdown     bool             // close by runtime error
}

type clientTimeout struct {
	client *Client
	err    error
}

type newCreateClient func(connection net.Conn, connectionInfo *server.ConnectionInfo) (client *Client, err error)

// implement IO's closer interface
var _ io.Closer = (*Client)(nil)

// Close the connection
func (client *Client) Close() error {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> Close: client %p is closing...", client)
	if client.isClosed {
		return errors.New("RPC client -> Close error: connection is already closed")
	} else {
		client.isClosed = true
		//log.Printf("RPC client -> Close: client %p is closed", client)
		return client.coder.Close()
	}
}

// IsAvailable check if the client still available
func (client *Client) IsAvailable() bool {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> IsAvailable: checking if client %p is available...", client)
	available := !client.isShutdown && !client.isClosed
	//log.Printf("RPC client -> IsAvailable: checked. client %p is available -> %t", client, available)
	return available
}

// addCall add new RPC call into the pendingCalls map
func (client *Client) addCall(call *Call) (uint64, error) {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> addCall: client %p is adding call %p...", client, call)
	if client.isClosed {
		return 0, errors.New("RPC client -> addCall error: client is closed")
	}
	if client.isShutdown {
		return 0, errors.New("RPC client -> addCall error: client is shutdown")
	}
	call.SequenceNumber = client.sequenceNumber
	client.pendingCalls[call.SequenceNumber] = call
	client.sequenceNumber++

	//actualStruct := reflect.ValueOf(call).Elem()
	//log.Printf("RPC client -> addCall: client %p added call %p with struct -> %+v to its pending call list", client, call, actualStruct)
	return call.SequenceNumber, nil
}

// deleteCall delete specific RPC call from pendingCalls map
func (client *Client) deleteCall(sequenceNumber uint64) *Call {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	call := client.pendingCalls[sequenceNumber]
	//log.Printf("RPC client -> deleteCall: client %p is deleting call %p from its pending call list...", client, call)
	delete(client.pendingCalls, sequenceNumber)
	//log.Printf("RPC client -> deleteCall: client %p deleted call %p from its pending call list", client, call)
	return call
}

// terminateCalls terminate all ROC calls in the pendingCalls map
func (client *Client) terminateCalls(Error error) {
	defer client.entityMutex.Unlock()
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> terminateCalls: client %p is terminating all calls in its pending call list...", client)
	client.isShutdown = true
	for _, call := range client.pendingCalls {
		call.Error = Error
		call.finishGo()
	}
	//log.Printf("RPC client -> terminateCalls: client %p terminated all calls in its pending call list", client)
}

// MakeDial enable client to connect to an RPC server
/*
func MakeDial(transportProtocol, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	//log.Printf("RPC client -> MakeDial: dialing and connecting to server %s...", serverAddress)
	connectionInfo, Error := parseConnectionInfo(connectionInfos...)
	if Error != nil {
		return nil, Error
	}
	connection, Error := net.Dial(transportProtocol, serverAddress)
	if Error != nil {
		return nil, Error
	}
	defer func() {
		if client == nil {
			_ = connection.Close()
		}
	}()
	//log.Printf("RPC client -> MakeDial: http handler %p successfully dialed and connected to server %s in protocol %s", connection, serverAddress, transportProtocol)
	return CreateClient(connection, connectionInfo)
}
*/

// XMakeDial calls specific Dial function to connect to an RPC server based on the first parameter of rpcAddress.
// rpcAddress is a general format (protocol@addr) to represent a rpc server
// eg, http@10.0.0.1:6666, tcp@10.0.0.1:7777, unix@/tmp/srpc.sock
func XMakeDial(rpcAddress string, connectionInfos ...*server.ConnectionInfo) (*Client, error) {
	parts := strings.Split(rpcAddress, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("RPC client -> XMakeDial err: wrong format for rpcAddress, expect 'protocol@addr', but got '%s'", rpcAddress)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return MakeDialHTTP("tcp", addr, connectionInfos...)
	default:
		// tcp, unix or other transport protocol
		return MakeDial(protocol, addr, connectionInfos...)
	}
}

func MakeDialHTTP(transportProtocol, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	//log.Printf("RPC client -> MakeDialHTTP: initializing HTTP dialing process...)
	return MakeDialWithTimeout(CreateClientHTTP, transportProtocol, serverAddress, connectionInfos...)
	//log.Printf("RPC client -> MakeDialHTTP: HTTP dialing process finished)
}

// CreateClientHTTP create a Client instance via HTTP transport protocol
func CreateClientHTTP(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	//log.Printf("RPC client -> CreateClientHTTP: creating an HTTP RPC client...")
	_, _ = io.WriteString(connection, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", server.DefaultRPCPath))

	// Require successful HTTP response before switching to RPC protocol.
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: "CONNECT"})
	if err == nil && response.Status == server.ConnectedMessage {
		return CreateClient(connection, connectionInfo)
	}
	if err == nil {
		err = errors.New("RPC client -> CreateClientHTTP error: unexpected HTTP response: " + response.Status)
	}
	return nil, err
}

// MakeDial enable client to connect to an RPC server
func MakeDial(transportProtocol, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	//log.Printf("RPC client -> MakeDial: initializing default dialing process...)
	return MakeDialWithTimeout(CreateClient, transportProtocol, serverAddress, connectionInfos...)
	//log.Printf("RPC client -> MakeDial: default dialing process finished)
}

// MakeDialWithTimeout enable client to connect to an RPC server within the configured timeout period
func MakeDialWithTimeout(createClient newCreateClient, transportProtocol, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	//log.Printf("RPC client -> MakeDialWithTimeout: dialing and connecting to server %s with timeout configuration %+v...", serverAddress, connectionInfos)
	connectionInfo, Error := parseConnectionInfo(connectionInfos...)
	if Error != nil {
		return nil, Error
	}
	connection, Error := net.DialTimeout(transportProtocol, serverAddress, connectionInfo.ConnectionTimeout)
	if Error != nil {
		return nil, Error
	}
	// close the connection immediately if there is any error
	defer func() {
		if Error != nil {
			_ = connection.Close()
		}
	}()
	//use go routine to create client
	timeoutChannel := make(chan clientTimeout)
	go func() {
		client, Error := createClient(connection, connectionInfo)
		timeoutChannel <- clientTimeout{client, Error}
	}()
	if connectionInfo.ConnectionTimeout == 0 {
		timeoutResult := <-timeoutChannel
		return timeoutResult.client, timeoutResult.err
	}
	//if time.After() channel receive message first, then createClient() operation is timeout, return error
	select {
	case <-time.After(connectionInfo.ConnectionTimeout):
		return nil, fmt.Errorf("RPC Client -> makeDialWithTimeout error: expect to connect server within %s, but connection timeout", connectionInfo.ConnectionTimeout)
	case timeoutResult := <-timeoutChannel:
		//log.Printf("RPC client -> MakeDial: http handler %p successfully dialed and connected to server %s in protocol %s", connection, serverAddress, transportProtocol)
		return timeoutResult.client, timeoutResult.err
	}
}

// parseConnectionInfo parse the input connection info to form proper connection info
func parseConnectionInfo(connectionInfos ...*server.ConnectionInfo) (*server.ConnectionInfo, error) {
	//log.Printf("RPC client -> parseConnectionInfo: parsing connectionInfo...")
	switch {
	case len(connectionInfos) == 0 || connectionInfos[0] == nil:
		//log.Printf("RPC client -> parseConnectionInfo: parsed nil...use server default connectionInfo")
		return server.DefaultConnectionInfo, nil
	case len(connectionInfos) != 1:
		return nil, errors.New("RPC Client -> parseConnectionInfo error: connectionInfos should be in length 1")
	default:
		connectionInfo := connectionInfos[0]
		connectionInfo.IDNumber = server.DefaultConnectionInfo.IDNumber
		if connectionInfo.CoderType == "" {
			connectionInfo.CoderType = server.DefaultConnectionInfo.CoderType
		}
		//log.Printf("RPC client -> parseConnectionInfo: parsed connectionInfo -> %+v", connectionInfo)
		return connectionInfo, nil
	}
}

// CreateClient create RPC client based on connection info
func CreateClient(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	//log.Printf("RPC client -> CreateClient: creating an RPC client...")
	coderFunction := coder.CoderInitializerMap[connectionInfo.CoderType]
	Error := json.NewEncoder(connection).Encode(connectionInfo)
	switch {
	case coderFunction == nil:
		err := fmt.Errorf("RPC Client -> CreateClient -> coderFunction error: %s coderType is invalid", connectionInfo.CoderType)
		//log.Println("RPC Client -> coder error:", err)
		return nil, err
	case Error != nil:
		//log.Println("RPC Client -> CreateClient -> connectionInfo error: ", Error)
		_ = connection.Close()
		return nil, Error
	default:
		mappedCoder := coderFunction(connection)
		client := &Client{
			sequenceNumber: 1,
			coder:          mappedCoder,
			connectionInfo: connectionInfo,
			pendingCalls:   make(map[uint64]*Call),
		}
		//log.Printf("RPC client -> CreateClient: RPC client %p created", client)
		go client.receiveCall()
		return client, nil
	}
}

// Call invokes the named function, waits for it to be finished, and returns its error status.
// Call also included timeout handling mechanism, which realized using context, so user can configure it themselves
// client-side timeout configure usage: context, _ := context.WithTimeout(context.Background(), time.Second)
func (client *Client) Call(serviceDotMethod string, inputs, output interface{}, context context.Context) error {
	log.Printf("RPC client -> Call: client %p invoking RPC request on function %s with inputs -> %v", client, serviceDotMethod, inputs)
	call := client.StartGo(serviceDotMethod, inputs, output, make(chan *Call, 1))
	select {
	case <-context.Done():
		client.deleteCall(call.SequenceNumber)
		return errors.New("RPC Client -> Call: client fail to finish the RPC request within the timeout period due to error: " + context.Err().Error())
	case call := <-call.Finish:
		//log.Printf("RPC client -> Call: client %p finished RPC request on function %s with inputs -> %v", client, serviceDotMethod, inputs)
		return call.Error
	}
}

// finishGo finish the RPC call and end the Go channel
func (call *Call) finishGo() {
	//log.Printf("RPC client -> finishGo: go channel %p finished the assigned call %p and destroyed", call.Finish, call)
	call.Finish <- call
}

// StartGo invokes the function asynchronously.
// It returns the Call structure representing the invocation
func (client *Client) StartGo(serviceDotMethod string, inputs, output interface{}, finish chan *Call) *Call {
	switch {
	case finish == nil:
		finish = make(chan *Call, 10)
	case cap(finish) == 0:
		log.Panic("RPC Client -> StartGo error: 'finishGo' channel is unbuffered")
	}
	call := &Call{
		ServiceDotMethod: serviceDotMethod,
		Inputs:           inputs,
		Output:           output,
		Finish:           finish,
	}
	//log.Printf("RPC client -> StartGo: go channel %p created and handling client %p call %p...", finish, client, call)
	client.sendCall(call)
	return call
}

// sendCall add RPC call to client's pending call list and send it to server
func (client *Client) sendCall(call *Call) {
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	//log.Printf("RPC client -> sendCall: client %p sending RPC request %p...", client, call)
	sequenceNumber, Error := client.addCall(call)
	if Error != nil {
		call.Error = Error
		call.finishGo()
		return
	}

	client.requestHeader.ServiceDotMethod = call.ServiceDotMethod
	client.requestHeader.SequenceNumber = sequenceNumber
	client.requestHeader.Error = ""
	requestHeader := &client.requestHeader
	requestBody := call.Inputs
	Error = client.coder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
	//log.Printf("RPC client -> sendCall: client %p sent RPC request %p", client, call)
	if Error != nil {
		if call := client.deleteCall(sequenceNumber); call != nil {
			call.Error = Error
			call.finishGo()
		}
	}
}

// receiveCall receive and decode any RPC response sent from server, and delete corresponding RPC call from client's pending call list
func (client *Client) receiveCall() {
	var Error error
	//log.Printf("RPC client -> receiveCall: client %p start listing on connection for all future RPC response...", client)
	for Error == nil {
		var response coder.MessageHeader
		if Error = client.coder.DecodeMessageHeader(&response); Error != nil {
			break
		}
		call := client.deleteCall(response.SequenceNumber)
		if call == nil {
			Error = client.coder.DecodeMessageBody(nil)
		} else if response.Error != "" {
			call.Error = fmt.Errorf(response.Error)
			Error = client.coder.DecodeMessageBody(nil)
			call.finishGo()
		} else {
			Error = client.coder.DecodeMessageBody(call.Output)
			if Error != nil {
				call.Error = errors.New("RPC Client -> receiveCall error: cannot decode message body" + Error.Error())
			}
			actualOutput := reflect.ValueOf(call.Output).Elem()
			log.Printf("RPC client -> receiveCall: client %p received call %p response -> %v", client, call, actualOutput)
			call.finishGo()
		}
	}
	client.terminateCalls(Error)
}
