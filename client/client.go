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
	clientAddress  string           // the local address of client
	isClosed       bool             // close by client
	isShutdown     bool             // close by runtime error
}

type clientTimeout struct {
	client *Client
	err    error
}

type CreateClientFunctionType func(connection net.Conn, connectionInfo *server.ConnectionInfo) (client *Client, err error)

// implement IO's closer interface
var _ io.Closer = (*Client)(nil)

// Close the connection
func (client *Client) Close() error {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> Close: RPC client %p is closing...", client)
	if client.isClosed {
		return errors.New("RPC client -> Close error: connection is already closed")
	} else {
		client.isClosed = true
		log.Printf("RPC client -> Close: RPC client %s is closed", client.clientAddress)
		return client.coder.Close()
	}
}

// IsAvailable check if the client still available
func (client *Client) IsAvailable() bool {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> IsAvailable: checking if RPC client %p is available...", client)
	available := !client.isShutdown && !client.isClosed
	log.Printf("RPC client -> IsAvailable: checked. RPC client %s availability is -> %t", client.clientAddress, available)
	return available
}

// addCall add new RPC call into the pendingCalls map
func (client *Client) addCall(call *Call) (uint64, error) {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> addCall: RPC client %p is adding call %p...", client, call)
	if client.isClosed {
		return 0, errors.New("RPC client -> addCall error: RPC client is closed")
	}
	if client.isShutdown {
		return 0, errors.New("RPC client -> addCall error: RPC client is shutdown")
	}
	call.SequenceNumber = client.sequenceNumber
	client.pendingCalls[call.SequenceNumber] = call
	client.sequenceNumber++

	//actualStruct := reflect.ValueOf(call).Elem()
	log.Printf("RPC client -> addCall: RPC client %s added RPC call %p to its pending call list", client.clientAddress, call)
	return call.SequenceNumber, nil
}

// deleteCall delete specific RPC call from pendingCalls map
func (client *Client) deleteCall(sequenceNumber uint64) *Call {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	call := client.pendingCalls[sequenceNumber]
	//log.Printf("RPC client -> deleteCall: RPC client %p is deleting call %p from its pending call list...", client, call)
	delete(client.pendingCalls, sequenceNumber)
	log.Printf("RPC client -> deleteCall: RPC client %s deleted RPC call %p from its pending call list", client.clientAddress, call)
	return call
}

// terminateCalls terminate all ROC calls in the pendingCalls map
func (client *Client) terminateCalls(Error error) {
	defer client.entityMutex.Unlock()
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	client.entityMutex.Lock()
	//log.Printf("RPC client -> terminateCalls: RPC client %p is terminating all calls in its pending call list...", client)
	client.isShutdown = true
	for _, call := range client.pendingCalls {
		call.Error = Error
		call.finishCall()
	}
	log.Printf("RPC client -> terminateCalls: RPC client %s terminated all RPC calls in its pending call list", client.clientAddress)
}

// finishCall finish the RPC call and end the Go channel
func (call *Call) finishCall() {
	log.Printf("RPC client -> finishCall: RPC call %p is finished, and passed its instance back to the coressponding Call() function", call)
	call.Finish <- call
}

// XMakeDial calls specific Dial function to connect to an RPC server based on the first parameter("http", "tcp", etc.) of completeServerAddress.
// completeServerAddress is a general format (protocol@addr) to represent a rpc server
// eg, http@10.0.0.1:6666, tcp@10.0.0.1:7777, unix@/tmp/srpc.sock
func XMakeDial(protocolAtServerAddress string, connectionInfos ...*server.ConnectionInfo) (*Client, error) {
	parts := strings.Split(protocolAtServerAddress, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("RPC client -> XMakeDial err: wrong format for protocolAtServerAddress, expect 'protocol@serverAddress', but got '%s'", protocolAtServerAddress)
	}
	protocol, serverAddress := parts[0], parts[1]
	switch protocol {
	case "http":
		//using tcp as transportation protocol, then encapsulate the communication using http application protocol
		log.Printf("RPC client -> XMakeDial: Dialing to RPC server %s, protocol is %s, triggered process MakeDialHTTP()...", protocolAtServerAddress, protocol)
		return MakeDialHTTP("tcp", serverAddress, connectionInfos...)
	default:
		// tcp, unix or other transportation protocol
		log.Printf("RPC client -> XMakeDial: Dialing to RPC server %s, protocol is %s, triggered process MakeDial()...", protocolAtServerAddress, protocol)
		return MakeDial(protocol, serverAddress, connectionInfos...)
	}
}

func MakeDialHTTP(transportProtocol string, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	log.Printf("RPC client -> MakeDialHTTP: initializing the HTTP dialing process MakeDialWithTimeout()...")
	return MakeDialWithTimeout(CreateClientHTTP, transportProtocol, serverAddress, connectionInfos...)
	//log.Printf("RPC client -> MakeDialHTTP: HTTP dialing process finished")
}

// CreateClientHTTP create a Client instance via HTTP application protocol
func CreateClientHTTP(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	log.Printf("RPC client -> CreateClientHTTP: creating an HTTP RPC client, sending HTTP CONNECT request to the RPC server...")
	_, _ = io.WriteString(connection, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", server.DefaultServerPath))

	// Require successful HTTP response before switching to RPC protocol.
	response, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: "CONNECT"})
	if err == nil && response.Status == server.ConnectedToServerMessage {
		log.Printf("RPC client -> CreateClientHTTP: recieved 200 HTTP response from the RPC server, initializing process CreateClient()...")
		return CreateClient(connection, connectionInfo)
	}
	if err == nil {
		err = errors.New("RPC client -> CreateClientHTTP error: unexpected HTTP response: " + response.Status)
	}
	return nil, err
}

// MakeDial enable RPC client to connect to an RPC server
func MakeDial(transportProtocol string, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	log.Printf("RPC client -> MakeDial: initializing the default dialing process MakeDialWithTimeout()...")
	return MakeDialWithTimeout(CreateClient, transportProtocol, serverAddress, connectionInfos...)
	//log.Printf("RPC client -> MakeDial: default dialing process finished")
}

// MakeDialWithTimeout enable client to connect to an RPC server within the configured timeout period
func MakeDialWithTimeout(createClient CreateClientFunctionType, transportProtocol string, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
	log.Printf("RPC client -> MakeDialWithTimeout: triggered process ParseConnectionInfo() to parss the connection info and DialTimeout() to dial and connect the RPC server...")
	connectionInfo, Error := ParseConnectionInfo(connectionInfos...)
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
	//using Go routine, create client based on the passed in CreateClientFunctionType(CreateClientHTTP or CreateClient)
	timeoutChannel := make(chan clientTimeout)
	go func() {
		newClient, err := createClient(connection, connectionInfo)
		timeoutChannel <- clientTimeout{newClient, err}
	}()
	if connectionInfo.ConnectionTimeout == 0 {
		timeoutResult := <-timeoutChannel
		return timeoutResult.client, timeoutResult.err
	}
	//if time.After() channel receive message first, then createClient() operation is timeout, return error
	select {
	case <-time.After(connectionInfo.ConnectionTimeout):
		return nil, fmt.Errorf("RPC Client -> makeDialWithTimeout error: expect to connect the RPC server within %s, but connection timeout", connectionInfo.ConnectionTimeout)
	case timeoutResult := <-timeoutChannel:
		log.Printf("RPC client -> MakeDialWithTimeout: successfully dialed and connected to RPC server %s in protocol %s", serverAddress, transportProtocol)
		return timeoutResult.client, timeoutResult.err
	}
}

// ParseConnectionInfo parse the input connection info to form proper connection info
func ParseConnectionInfo(connectionInfos ...*server.ConnectionInfo) (*server.ConnectionInfo, error) {
	//log.Printf("RPC client -> ParseConnectionInfo: parsing connectionInfo...")
	switch {
	case len(connectionInfos) == 0 || connectionInfos[0] == nil:
		log.Printf("RPC client -> ParseConnectionInfo: parsed nil, uses server default connectionInfo -> %+v", server.DefaultConnectionInfo)
		return server.DefaultConnectionInfo, nil
	case len(connectionInfos) != 1:
		return nil, errors.New("RPC client -> ParseConnectionInfo error: connectionInfos should be in length 1")
	default:
		connectionInfo := connectionInfos[0]
		connectionInfo.IDNumber = server.DefaultConnectionInfo.IDNumber
		if connectionInfo.CoderType == "" {
			connectionInfo.CoderType = server.DefaultConnectionInfo.CoderType
		}
		log.Printf("RPC client -> ParseConnectionInfo: parsed connectionInfo -> %+v", connectionInfo)
		return connectionInfo, nil
	}
}

// CreateClient create RPC client based on connection info
func CreateClient(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	log.Printf("RPC client -> CreateClient: creating an RPC client...")
	coderFunction := coder.CoderInitializerMap[connectionInfo.CoderType]
	Error := json.NewEncoder(connection).Encode(connectionInfo)
	switch {
	case coderFunction == nil:
		err := fmt.Errorf("RPC client -> CreateClient error: coderFunction error: %s coderType is invalid", connectionInfo.CoderType)
		return nil, err
	case Error != nil:
		log.Println("RPC client -> CreateClient error: connectionInfo error: ", Error)
		_ = connection.Close()
		return nil, Error
	default:
		mappedCoder := coderFunction(connection)
		client := &Client{
			sequenceNumber: 1,
			coder:          mappedCoder,
			connectionInfo: connectionInfo,
			pendingCalls:   make(map[uint64]*Call),
			clientAddress:  connection.LocalAddr().Network() + "@" + connection.LocalAddr().String(),
		}
		log.Printf("RPC client -> CreateClient: RPC client %s created with field -> %+v", client.clientAddress, client)
		log.Printf("RPC client -> CreateClient: initializing sub Go routine to actively receive all future RPC call response...")
		go client.receiveCall()
		return client, nil
	}
}

// Call invokes the named function, waits for it to be finished, and returns its error status.
// Call also included timeout handling mechanism, which realized using context, so user can configure it themselves
// client-side timeout configure usage: context, _ := context.WithTimeout(context.Background(), time.Second)
func (client *Client) Call(serviceDotMethod string, inputs, output interface{}, context context.Context) error {
	log.Printf("RPC client -> Call: RPC client %s invoking RPC call on function %s with inputs -> %+v", client.clientAddress, serviceDotMethod, inputs)
	call := client.startCall(serviceDotMethod, inputs, output, make(chan *Call, 1))
	select {
	case <-context.Done():
		client.deleteCall(call.SequenceNumber)
		return errors.New("RPC client -> Call: RPC client fail to finish the RPC request within the timeout period due to error: " + context.Err().Error())
	case finishedCall := <-call.Finish:
		//log.Printf("RPC client -> Call: RPC client %p finished RPC request on function %s with inputs -> %+v", client, serviceDotMethod, inputs)
		log.Printf("RPC client -> Call: RPC client %s finished RPC call %p with field -> %+v", client.clientAddress, call, call)
		return finishedCall.Error
	}
}

// startCall invokes the function asynchronously.
// It returns the Call structure representing the invocation
func (client *Client) startCall(serviceDotMethod string, inputs, output interface{}, finish chan *Call) *Call {
	switch {
	case finish == nil:
		finish = make(chan *Call, 10)
	case cap(finish) == 0:
		log.Panic("RPC client -> startCall error: 'finishCall' channel is unbuffered")
	}
	call := &Call{
		ServiceDotMethod: serviceDotMethod,
		Inputs:           inputs,
		Output:           output,
		Finish:           finish,
	}
	log.Printf("RPC client -> startCall: RPC client %s constructed RPC call %p with field -> %+v", client.clientAddress, call, call)
	log.Printf("RPC client -> startCall: initializaing process sendCall()...")
	client.sendCall(call)
	return call
}

// sendCall add RPC call to client's pending call list and send it to server
func (client *Client) sendCall(call *Call) {
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	//log.Printf("RPC client -> sendCall: RPC client %p sending RPC request %p...", client, call)
	sequenceNumber, Error := client.addCall(call)
	if Error != nil {
		call.Error = Error
		call.finishCall()
		return
	}

	client.requestHeader.ServiceDotMethod = call.ServiceDotMethod
	client.requestHeader.SequenceNumber = sequenceNumber
	client.requestHeader.Error = ""
	requestHeader := &client.requestHeader
	requestBody := call.Inputs
	//log.Printf("RPC client -> sendCall: RPC client %p sent RPC request %+v", client, call)
	log.Printf("RPC client -> sendCall: RPC client %s sending RPC call %p...", client.clientAddress, call)
	Error = client.coder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
	if Error != nil {
		if matchedCall := client.deleteCall(sequenceNumber); matchedCall != nil {
			matchedCall.Error = Error
			matchedCall.finishCall()
		}
	}
}

// receiveCall receive and decode any RPC response sent from server, and delete corresponding RPC call from client's pending call list
func (client *Client) receiveCall() {
	var err error
	for err == nil {
		var response coder.MessageHeader
		log.Printf("RPC client -> receiveCall: RPC client %s recieved response from the RPC server, initializaing process DecodeMessageHeader() to decode the message header...", client.clientAddress)
		if err = client.coder.DecodeMessageHeader(&response); err != nil {
			log.Printf("RPC client -> receiveCall error: DecodeMessageHeader failed due to error: %s", err.Error())
			break
		}
		log.Printf("RPC client -> receiveCall: RPC client %s finished decoding message header, initializaing process deleteCall() to fetch(and delete) the coressponding RPC call...", client.clientAddress)
		call := client.deleteCall(response.SequenceNumber)
		if call == nil {
			log.Printf("RPC client -> receiveCall error: RPC client %s fetch RPC call nil, DecodeMessageBody() and dumped the message...", client.clientAddress)
			err = client.coder.DecodeMessageBody(nil)
		} else if response.Error != "" {
			call.Error = fmt.Errorf(response.Error)
			log.Printf("RPC client -> receiveCall error: RPC client %s fetch RPC call %p with reponse error %s, DecodeMessageBody() and finishCall() with error...", client.clientAddress, call, call.Error)
			err = client.coder.DecodeMessageBody(nil)
			call.finishCall()
		} else {
			log.Printf("RPC client -> receiveCall: RPC client %s finished fetching the coressponding RPC call, initializaing process DecodeMessageBody() to decode the message body...", client.clientAddress)
			err = client.coder.DecodeMessageBody(call.Output)
			if err != nil {
				call.Error = errors.New("RPC client -> receiveCall error: DecodeMessageBody() fail to decode message body" + err.Error())
			}
			//actualOutput := reflect.ValueOf(call.Output).Elem()
			//log.Printf("RPC client -> receiveCall: RPC client %p received response from RPC request %+v with output -> %+v", client, call, call.Output)
			log.Printf("RPC client -> receiveCall: RPC client %s finished decoding the message, received response from RPC call %p with field %+v and finishing it...", client.clientAddress, call, call)
			call.finishCall()
		}
	}
	client.terminateCalls(err)
}
