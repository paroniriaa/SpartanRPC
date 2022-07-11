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
	Finish           chan *Call  // Receives *Call when the call that Go channels handle is finished
}

// Client represents an RPC Client.
type Client struct {
	coder          coder.Coder            // the message coder for message encoding and decoding
	connectionInfo *server.ConnectionInfo // the connection (between server and client ) info

	requestMutex  sync.Mutex    // protects following
	requestHeader coder.Message // the header of an RPC request

	entityMutex    sync.Mutex       // protects following
	sequenceNumber uint64           // the sequence number that counts the RPC request
	pendingCalls   map[uint64]*Call // the map that store all pending RPC calls, key is Call.SequenceNumber, value is Call
	isClosed       bool             // close by client
	isShutdown     bool             // close by runtime error
}

// implement IO's closer interface
var _ io.Closer = (*Client)(nil)

// Close the connection
func (client *Client) Close() error {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	log.Printf("RPC client -> Close: client %p is closing...", client)
	if client.isClosed {
		return errors.New("RPC client -> Close error: connection is already closed")
	} else {
		client.isClosed = true
		log.Printf("RPC client -> Close: client %p is closed", client)
		return client.coder.Close()
	}
}

// IsAvailable check if the client still available
func (client *Client) IsAvailable() bool {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	log.Printf("RPC client -> IsAvailable: checking if client %p is available...", client)
	available := !client.isShutdown && !client.isClosed
	log.Printf("RPC client -> IsAvailable: checked. client %p is available -> %t", client, available)
	return available
}

// addCall add new RPC call into the pendingCalls map
func (client *Client) addCall(call *Call) (uint64, error) {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	log.Printf("RPC client -> addCall: client %p is adding call %p...", client, call)
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
	log.Printf("RPC client -> addCall: client %p added call %p with struct -> %+v to its pending call list", client, call, actualStruct)
	return call.SequenceNumber, nil
}

// deleteCall delete specific RPC call from pendingCalls map
func (client *Client) deleteCall(sequenceNumber uint64) *Call {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	call := client.pendingCalls[sequenceNumber]
	log.Printf("RPC client -> deleteCall: client %p is deleting call %p from its pending call list...", client, call)
	delete(client.pendingCalls, sequenceNumber)
	log.Printf("RPC client -> deleteCall: client %p deleted call %p from its pending call list", client, call)
	return call
}

// terminateCalls terminate all ROC calls in the pendingCalls map
func (client *Client) terminateCalls(Error error) {
	defer client.entityMutex.Unlock()
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	client.entityMutex.Lock()
	log.Printf("RPC client -> terminateCalls: client %p is terminating all calls in its pending call list...", client)
	client.isShutdown = true
	for _, call := range client.pendingCalls {
		call.Error = Error
		call.finishGo()
	}
	log.Printf("RPC client -> terminateCalls: client %p terminated all calls in its pending call list", client)
}

// CreateClient create RPC client based on connection info
func CreateClient(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	log.Printf("RPC client -> CreateClient: creating an RPC client...")
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
		log.Printf("RPC client -> CreateClient: RPC client %p created", client)
		go client.receiveCall()
		return client, nil
	}
}

// parseConnectionInfo parse the input connection info to form proper connection info
func parseConnectionInfo(connectionInfos ...*server.ConnectionInfo) (*server.ConnectionInfo, error) {
	log.Printf("RPC client -> parseConnectionInfo: parsing connectionInfo...")
	switch {
	case len(connectionInfos) == 0 || connectionInfos[0] == nil:
		log.Printf("RPC client -> parseConnectionInfo: parsed nil...use server default connectionInfo")
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
	log.Printf("RPC client -> MakeDial: dialing and conneting to server %s...", serverAddress)
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
	log.Printf("RPC client -> MakeDial: http handler %p successfuly dialed and connected to server %s in protocol %s", connection, serverAddress, transportProtocol)
	return CreateClient(connection, connectionInfo)
}

// Call invokes the named function, waits for it to be finished, and returns its error status.
func (client *Client) Call(serviceDotMethod string, inputs, output interface{}) error {
	log.Printf("RPC client -> Call: client %p invoking RPC request on function %s with inputs -> %v", client, serviceDotMethod, inputs)
	call := <-client.startGo(serviceDotMethod, inputs, output, make(chan *Call, 1)).Finish
	log.Printf("RPC client -> Call: client %p finished RPC reuqest on function %s with inputs -> %v", client, serviceDotMethod, inputs)
	return call.Error
}

// finishGo finish the RPC call and end the Go channel
func (call *Call) finishGo() {
	log.Printf("RPC client -> finishGo: go channel %p finished the assigned call %p and destoryed", call.Finish, call)
	call.Finish <- call
}

// startGo invokes the function asynchronously.
// It returns the Call structure representing the invocation
func (client *Client) startGo(serviceDotMethod string, inputs, output interface{}, finish chan *Call) *Call {
	switch {
	case finish == nil:
		finish = make(chan *Call, 10)
	case cap(finish) == 0:
		log.Panic("RPC Client -> startGo error: 'finishGo' channel is unbuffered")
	}
	call := &Call{
		ServiceDotMethod: serviceDotMethod,
		Inputs:           inputs,
		Output:           output,
		Finish:           finish,
	}
	log.Printf("RPC client -> startGo: go channel %p created and handiling client %p call %p...", finish, client, call)
	client.sendCall(call)
	return call
}

// sendCall add RPC call to client's pending call list and send it to server
func (client *Client) sendCall(call *Call) {
	defer client.requestMutex.Unlock()
	client.requestMutex.Lock()
	log.Printf("RPC client -> sendCall: client %p sending RPC request %p...", client, call)
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
	log.Printf("RPC client -> sendCall: client %p sent RPC request %p", client, call)
	if Error != nil {
		log.Print("---------")
		if call := client.deleteCall(sequenceNumber); call != nil {
			call.Error = Error
			call.finishGo()
		}
	}
}

// receiveCall receive and decode any RPC response sent from server, and delete corresponding RPC call from client's pending call list
func (client *Client) receiveCall() {
	var Error error
	log.Printf("RPC client -> receiveCall: client %p start listeing on connection for all future RPC response...", client)
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
			call.finishGo()
		} else {
			Error = client.coder.DecodeMessageBody(call.Output)
			if Error != nil {
				call.Error = errors.New("RPC Client -> receiveCall error: cannot decode message body" + Error.Error())
			}
			actualOutput := reflect.ValueOf(call.Output).Elem()
			log.Printf("RPC client -> receiveCall: client %p recieved call %p response -> %+v", client, call, actualOutput)
			call.finishGo()
		}
	}
	client.terminateCalls(Error)
}
