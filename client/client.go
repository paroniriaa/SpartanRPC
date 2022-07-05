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
	SeqNumber     uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

type Client struct {
	message      coder.Coder
	option       *server.Option
	sending      sync.Mutex
	header       coder.Header
	locker       sync.Mutex
	seqNumber    uint64
	pending_Pool map[uint64]*Call
	IsClosed     bool
	IsShutdown   bool
}

//TODO: variable
var _ io.Closer = (*Client)(nil)

//TODO: function
func (call *Call) done() {
	call.Done <- call
}

func (client *Client) Close() error {
	defer client.locker.Unlock()
	client.locker.Lock()
	if client.IsClosed {
		return errors.New("RPC client - close error: connection is closed")
	} else {
		client.IsClosed = true
		return client.message.Close()
	}
}

func (client *Client) IsAvailable() bool {
	defer client.locker.Unlock()
	client.locker.Lock()
	return !client.IsShutdown && !client.IsClosed
}

func (client *Client) addCall(call *Call) (uint64, error) {
	client.locker.Lock()
	defer client.locker.Unlock()
	if client.IsClosed {
		return 0, errors.New("RPC client - addCall error: client is closed")
	}
	if client.IsShutdown {
		return 0, errors.New("RPC client - addCall error: client is shutdown")
	}
	call.SeqNumber = client.seqNumber
	client.pending_Pool[call.SeqNumber] = call
	client.seqNumber++
	return call.SeqNumber, nil
}

func (client *Client) deleteCall(seqNumber uint64) *Call {
	client.locker.Lock()
	defer client.locker.Unlock()
	call := client.pending_Pool[seqNumber]
	delete(client.pending_Pool, seqNumber)
	return call
}

func (client *Client) terminateCalls(Error error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.locker.Lock()
	defer client.locker.Unlock()
	client.IsShutdown = true
	for _, call := range client.pending_Pool {
		call.Error = Error
		call.done()
	}
}

func (client *Client) receiveCall() {
	var Error error
	for Error == nil {
		var h coder.Header
		if Error = client.message.DecodeMessageHeader(&h); Error != nil {
			break
		}
		call := client.deleteCall(h.SequenceNumber)
		if call == nil {
			Error = client.message.DecodeMessageBody(nil)
		} else if h.Error != "" {
			call.Error = fmt.Errorf(h.Error)
			Error = client.message.DecodeMessageBody(nil)
			call.done()
		} else {
			Error = client.message.DecodeMessageBody(call.Reply)
			if Error != nil {
				call.Error = errors.New("RPC Client - receiveCall error: reading body " + Error.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(Error)
}

func CreateClient(connection net.Conn, option *server.Option) (*Client, error) {
	functionMap := coder.CoderFunctionMap[option.CoderType]
	errors := json.NewEncoder(connection).Encode(option)
	switch {
	case functionMap == nil:
		err := fmt.Errorf("RPC Client - createClient error: %s coderType is invalid", option.CoderType)
		log.Println("RPC Client - coder error:", err)
		return nil, err
	case errors != nil:
		log.Println("RPC Client - option error: ", errors)
		_ = connection.Close()
		return nil, errors
	default:
		return newClientCoder(functionMap(connection), option), nil
	}
}

func newClientCoder(message coder.Coder, option *server.Option) *Client {
	client := &Client{
		seqNumber:    1,
		message:      message,
		option:       option,
		pending_Pool: make(map[uint64]*Call),
	}
	go client.receiveCall()
	return client
}

func parseOptions(options ...*server.Option) (*server.Option, error) {
	switch {
	case len(options) == 0 || options[0] == nil:
		return server.DefaultOption, nil
	case len(options) != 1:
		return nil, errors.New("RPC Client - parseOptions error: options should be lower than 2")
	default:
		opt := options[0]
		opt.IDNumber = server.DefaultOption.IDNumber
		if opt.CoderType == "" {
			opt.CoderType = server.DefaultOption.CoderType
		}
		return opt, nil
	}
}

func MakeDial(protocol, serverAddress string, options ...*server.Option) (client *Client, err error) {
	opt, err := parseOptions(options...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(protocol, serverAddress)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return CreateClient(conn, opt)
}

func (client *Client) sendCall(call *Call) {
	defer client.sending.Unlock()
	client.sending.Lock()
	seqNumber, err := client.addCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	client.header.ServiceMethod = call.ServiceMethod
	client.header.SequenceNumber = seqNumber
	client.header.Error = ""
	Error := client.message.EncodeMessageHeaderAndBody(&client.header, call.Args)
	if Error != nil {
		if call := client.deleteCall(seqNumber); call != nil {
			call.Error = Error
			call.done()
		}
	}
}

func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	switch {
	case done == nil:
		done = make(chan *Call, 10)
	case cap(done) == 0:
		log.Panic("RPC Client - Go error: unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.sendCall(call)
	return call
}

func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}
