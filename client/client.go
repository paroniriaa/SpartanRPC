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
	"sync"
)

// TODO: struct
type Call struct {
	SequenceNumber   uint64
	ServiceDotMethod string
	Inputs           interface{}
	Output           interface{}
	Error            error
	Complete         chan *Call
}

type Client struct {
	coder          coder.Coder
	connectionInfo *server.ConnectionInfo
	sendingMutex   sync.Mutex
	header         coder.Header
	entityMutex    sync.Mutex
	sequenceNumber uint64
	callMap        map[uint64]*Call
	isClosed       bool
	isShutdown     bool
}

//TODO: variable
var _ io.Closer = (*Client)(nil)

//TODO: function
func (call *Call) complete() {
	call.Complete <- call
}

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
	client.callMap[call.SequenceNumber] = call
	client.sequenceNumber++
	return call.SequenceNumber, nil
}

func (client *Client) deleteCall(sequenceNumber uint64) *Call {
	client.entityMutex.Lock()
	defer client.entityMutex.Unlock()
	call := client.callMap[sequenceNumber]
	delete(client.callMap, sequenceNumber)
	return call
}

func (client *Client) terminateCalls(Error error) {
	client.sendingMutex.Lock()
	defer client.sendingMutex.Unlock()
	client.entityMutex.Lock()
	defer client.entityMutex.Unlock()
	client.isShutdown = true
	for _, call := range client.callMap {
		call.Error = Error
		call.complete()
	}
}

func (client *Client) receiveCall() {
	var Error error
	for Error == nil {
		var h coder.Header
		if Error = client.coder.DecodeMessageHeader(&h); Error != nil {
			break
		}
		call := client.deleteCall(h.SequenceNumber)
		if call == nil {
			Error = client.coder.DecodeMessageBody(nil)
		} else if h.Error != "" {
			call.Error = fmt.Errorf(h.Error)
			Error = client.coder.DecodeMessageBody(nil)
			call.complete()
		} else {
			Error = client.coder.DecodeMessageBody(call.Output)
			if Error != nil {
				call.Error = errors.New("RPC Client -> receiveCall error: cannot decode message body " + Error.Error())
			}
			call.complete()
		}
	}
	client.terminateCalls(Error)
}

func CreateClient(connection net.Conn, connectionInfo *server.ConnectionInfo) (*Client, error) {
	coderFunction := coder.CoderFunctionMap[connectionInfo.CoderType]
	Error := json.NewEncoder(connection).Encode(connectionInfo)
	switch {
	case coderFunction == nil:
		err := fmt.Errorf("RPC Client -> createClient error: %s coderType is invalid", connectionInfo.CoderType)
		log.Println("RPC Client -> coder error:", err)
		return nil, err
	case Error != nil:
		log.Println("RPC Client -> connectionInfo error: ", Error)
		_ = connection.Close()
		return nil, Error
	default:
		mappedCoder := coderFunction(connection)
		client := &Client{
			sequenceNumber: 1,
			coder:          mappedCoder,
			connectionInfo: connectionInfo,
			callMap:        make(map[uint64]*Call),
		}
		go client.receiveCall()
		return client, nil
	}
}

func parseConnectionInfo(connectionInfos ...*server.ConnectionInfo) (*server.ConnectionInfo, error) {
	switch {
	case len(connectionInfos) == 0 || connectionInfos[0] == nil:
		return server.DefaultConnectionInfo, nil
	case len(connectionInfos) != 1:
		return nil, errors.New("RPC Client -> parseConnectionInfo error: connectionInfos should be in length 1")
	default:
		connectionInfo := connectionInfos[0]
		connectionInfo.IDNumber = server.DefaultConnectionInfo.IDNumber
		if connectionInfo.CoderType == "" {
			connectionInfo.CoderType = server.DefaultConnectionInfo.CoderType
		}
		return connectionInfo, nil
	}
}

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
	return CreateClient(connection, connectionInfo)
}

func (client *Client) sendCall(call *Call) {
	defer client.sendingMutex.Unlock()
	client.sendingMutex.Lock()
	sequenceNumber, Error := client.addCall(call)
	if Error != nil {
		call.Error = Error
		call.complete()
		return
	}

	client.header.ServiceDotMethod = call.ServiceDotMethod
	client.header.SequenceNumber = sequenceNumber
	client.header.Error = ""
	Error = client.coder.EncodeMessageHeaderAndBody(&client.header, call.Inputs)
	if Error != nil {
		if call := client.deleteCall(sequenceNumber); call != nil {
			call.Error = Error
			call.complete()
		}
	}
}

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
	client.sendCall(call)
	return call
}

func (client *Client) Call(serviceDotMethod string, inputs, output interface{}) error {
	call := <-client.Go(serviceDotMethod, inputs, output, make(chan *Call, 1)).Complete
	return call.Error
}

func (client *Client) Close() error {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	if client.isClosed {
		return errors.New("RPC client -> Close error: connection is shutdown")
	} else {
		client.isClosed = true
		return client.coder.Close()
	}
}

func (client *Client) IsAvailable() bool {
	defer client.entityMutex.Unlock()
	client.entityMutex.Lock()
	return !client.isShutdown && !client.isClosed
}
