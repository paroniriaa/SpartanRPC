package client

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

// Call represents an active RPC.
type Call struct {
	SequenceNumber   uint64      // the sequence number of the RPC call
	ServiceDotMethod string      // the service and method to call (format: service.method)
	Inputs           interface{} // the inputs to the function (*struct)
	Output           interface{} // The output from the function (*struct).
	Error            error       // After completion, the error status.
	Complete         chan *Call  // Receives *Call when Go is complete
}

// Client represents an RPC Client.
type Client struct {
	coder          coder.Coder            // the message coder for message encoding and decoding
	connectionInfo *server.ConnectionInfo // the connection (between server and client ) info

	requestMutex  sync.Mutex    // protects following
	requestHeader coder.Message // the header of an RPC request

	entityMutex    sync.Mutex       // protects following
	sequenceNumber uint64           // the sequence number of the RPC client
	pendingCalls   map[uint64]*Call // the map that store all pending RPC calls, key is Call.SequenceNumber, value is Call
	isClosed       bool             // close by client
	isShutdown     bool             // close by runtime error
}

// implement IO's closer interface
var _ io.Closer = (*Client)(nil)

// complete the RPC call
func (call *Call) complete() {
	call.Complete <- call
	log.Printf("RPC call -> complete: call #%d has completed", call.SequenceNumber)
}

// Close the connection
func (client *Client) Close() error {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	log.Printf("RPC client -> Close: client %p is closed", client)
	if client.isClosed {
		return errors.New("RPC client -> Close error: connection is already closed")
	} else {
		client.isClosed = true
		return client.coder.Close()
	}
}

// IsAvailable check if the client still available
func (client *Client) IsAvailable() bool {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	available := !client.isShutdown && !client.isClosed
	log.Printf("RPC client -> IsAvailable: checking if client %p is available -> %t", client, available)
	return available
}

// addCall add new RPC call into the pendingCalls map
func (client *Client) addCall(call *Call) (uint64, error) {
	client.entityMutex.Lock()
	defer client.entityMutex.Unlock()
	if client.isClosed {
		return 0, errors.New("RPC client -> addCall error: client is closed")
	}
	if client.isShutdown {
		return 0, errors.New("RPC client -> addCall error: client is shutdown")
	}
	call.SequenceNumber = client.sequenceNumber
	client.pendingCalls[call.SequenceNumber] = call
	client.sequenceNumber++

	actualStruct := reflect.ValueOf(call).Elem()
	log.Printf("RPC client -> addCall: client %p added call #%d with struct %+v to its pending call list", client, call.SequenceNumber, actualStruct)
	return call.SequenceNumber, nil
}

// deleteCall delete specific RPC call from pendingCalls map
func (client *Client) deleteCall(sequenceNumber uint64) *Call {
	client.entityMutex.Lock()
	defer client.entityMutex.Unlock()
	call := client.pendingCalls[sequenceNumber]
	log.Printf("RPC client -> deleteCall: client %p deleted call #%d from its pending call list", client, call.SequenceNumber)
	delete(client.pendingCalls, sequenceNumber)
	return call
}

// terminateCalls terminate all ROC calls in the pendingCalls map
func (client *Client) terminateCalls(Error error) {
	client.requestMutex.Lock()
	defer client.requestMutex.Unlock()
	client.entityMutex.Lock()
	defer client.entityMutex.Unlock()
	client.isShutdown = true
	for _, call := range client.pendingCalls {
		call.Error = Error
		call.complete()
	}
	log.Printf("RPC client -> terminateCalls: client %p terminated all calls in its pending call list", client)
}

// CreateClient create RPC client based on connection info
func CreateClient(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	coderFunction := coder.CoderFunctionMap[connectionInfo.CoderType]
	Error := json.NewEncoder(connection).Encode(connectionInfo)
	switch {
	case coderFunction == nil:
		err := fmt.Errorf("RPC Client -> CreateClient -> coderFunction error: %s coderType is invalid", connectionInfo.CoderType)
		log.Println("RPC Client -> coder error:", err)
		return nil, err
	case Error != nil:
		log.Println("RPC Client -> CreateClient -> connectionInfo error: ", Error)
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
		log.Printf("RPC client -> CreateClient: client %p created", client)
		go client.receiveCall()
		return client, nil
	}
}

// parseConnectionInfo parse the input connection info to form proper connection info
func parseConnectionInfo(connectionInfos ...*server.ConnectionInfo) (*server.ConnectionInfo, error) {
	switch {
	case len(connectionInfos) == 0 || connectionInfos[0] == nil:
		log.Printf("RPC client -> parseConnectionInfo: parsed nil from arguments when dialing...use server default connectionInfo")
		return server.DefaultConnectionInfo, nil
	case len(connectionInfos) != 1:
		return nil, errors.New("RPC Client -> parseConnectionInfo error: connectionInfos should be in length 1")
	default:
		connectionInfo := connectionInfos[0]
		connectionInfo.IDNumber = server.DefaultConnectionInfo.IDNumber
		if connectionInfo.CoderType == "" {
			connectionInfo.CoderType = server.DefaultConnectionInfo.CoderType
		}
		log.Printf("RPC client -> parseConnectionInfo: parsed connectionInfo -> %+v", connectionInfo)
		return connectionInfo, nil
	}
}

// MakeDial enable client to connect to an RPC server
func MakeDial(transportProtocol, serverAddress string, connectionInfos ...*server.ConnectionInfo) (client *Client, Error error) {
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
	log.Printf("RPC client -> MakeDial: http handler %p dialed server %s in protocol %s...success", &connection, serverAddress, transportProtocol)
	return CreateClient(connection, connectionInfo)
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (client *Client) Call(serviceDotMethod string, inputs, output interface{}) error {
	log.Printf("RPC client -> Call: client %p invokes RPC call %s with inputs -> %v", client, serviceDotMethod, inputs)
	call := <-client.Go(serviceDotMethod, inputs, output, make(chan *Call, 1)).Complete
	return call.Error
}

// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation
func (client *Client) Go(serviceDotMethod string, inputs, output interface{}, complete chan *Call) *Call {
	switch {
	case complete == nil:
		complete = make(chan *Call, 10)
	case cap(complete) == 0:
		log.Panic("RPC Client -> Go error: 'complete' channel is unbuffered")
	}
	call := &Call{
		ServiceDotMethod: serviceDotMethod,
		Inputs:           inputs,
		Output:           output,
		Complete:         complete,
	}
	log.Printf("RPC client -> Go: go channel %p is asynchronously handiling client %p RPC call", complete, client)
	client.sendCall(call)
	return call
}

// sendCall make request to a RPC call
func (client *Client) sendCall(call *Call) {
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	sequenceNumber, Error := client.addCall(call)
	if Error != nil {
		call.Error = Error
		call.complete()
		return
	}

	client.requestHeader.ServiceDotMethod = call.ServiceDotMethod
	client.requestHeader.SequenceNumber = sequenceNumber
	client.requestHeader.Error = ""
	requestHeader := &client.requestHeader
	requestBody := call.Inputs
	log.Printf("RPC client -> sendCall: client %p sent RPC request #%d", client, call.SequenceNumber)
	Error = client.coder.EncodeMessageHeaderAndBody(requestHeader, requestBody)
	if Error != nil {
		if call := client.deleteCall(sequenceNumber); call != nil {
			call.Error = Error
			call.complete()
		}
	}
}

// receiveCall receive response from the RPC call
func (client *Client) receiveCall() {
	var Error error
	for Error == nil {
		var response coder.Message
		if Error = client.coder.DecodeMessageHeader(&response); Error != nil {
			break
		}
		call := client.deleteCall(response.SequenceNumber)
		if call == nil {
			Error = client.coder.DecodeMessageBody(nil)
		} else if response.Error != "" {
			call.Error = fmt.Errorf(response.Error)
			Error = client.coder.DecodeMessageBody(nil)
			call.complete()
		} else {
			Error = client.coder.DecodeMessageBody(call.Output)
			if Error != nil {
				call.Error = errors.New("RPC Client -> receiveCall error: cannot decode message body" + Error.Error())
			}
			actualOutput := reflect.ValueOf(call.Output).Elem()
			log.Printf("RPC client -> receiveCall: client %p recieved call #%d response -> %+v", client, call.SequenceNumber, actualOutput)
			call.complete()
		}
	}
	client.terminateCalls(Error)
}
