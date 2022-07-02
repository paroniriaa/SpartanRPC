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

// Call represents an active RPC.
type Call struct {
	SeqNumber           uint64
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // arguments to the function
	Reply         interface{} // reply from the function
	Error         error       // if error occurs, it will be set
	Done          chan *Call  // Strobes when call is complete.
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	message       coder.Coder
	option      *server.Option
	sending  sync.Mutex // protect following
	header   coder.Header
	locker       sync.Mutex // protect following
	seqNumber      uint64
	pending_Pool  map[uint64]*Call
	IsClosed  bool // user has called Close
	IsShutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil)

// Close the connection
func (client *Client) Close() error {
	defer client.locker.Unlock()
	client.locker.Lock()
	if client.IsClosed {
		return errors.New("connection is shut down")
	} else {
		client.IsClosed = true
		return client.message.Close()
	}
}

// IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	defer client.locker.Unlock()
	client.locker.Lock()
	return !client.IsShutdown && !client.IsClosed
}

func (client *Client) addCall(call *Call) (uint64, error) {
	client.locker.Lock()
	defer client.locker.Unlock()
	if client.IsClosed{
		if client.IsShutdown {
			return 0, errors.New("connection is shut down")
		}
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

// for loop
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
		if Error = client.message.ReadHeader(&h); Error != nil {
			break
		}
		call := client.deleteCall(h.SequenceNumber)
		if call == nil {
			Error = client.message.ReadBody(nil)
		} else if h.Error != "" {
			call.Error = fmt.Errorf(h.Error)
			Error = client.message.ReadBody(nil)
			call.done()
		} else{
			Error = client.message.ReadBody(call.Reply)
			if Error != nil {
				call.Error = errors.New("reading body " + Error.Error())
			}
			call.done()
		}
	}
	// error occurs, so terminateCalls pending calls
	client.terminateCalls(Error)
}

func CreateClient(connection net.Conn, option *server.Option) (*Client, error) {
	functionMap := coder.NewCoderFuncMap[option.CoderType]
	errors := json.NewEncoder(connection).Encode(option)
	switch{
	case functionMap == nil:
		err := fmt.Errorf("invalid codec type %s", option.CoderType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	case errors != nil:
		log.Println("rpc client: options error: ", errors)
		_ = connection.Close()
		return nil, errors
	default:
		return newClientCoder(functionMap(connection), option), nil
	}
}

func newClientCoder(message coder.Coder, option *server.Option) *Client {
	client := &Client{
		seqNumber:     1, // seq starts with 1, 0 means invalid call
		message:      message,
		option:     option,
		pending_Pool: make(map[uint64]*Call),
	}
	go client.receiveCall()
	return client
}

func parseOptions(options ...*server.Option) (*server.Option, error) {
	// if opts is nil or pass nil as parameter
	switch{
	case len(options) == 0 || options[0] == nil:
		return server.DefaultOption, nil
	case len(options) != 1:
		return nil, errors.New("number of options is more than 1")
	default:
		opt := options[0]
		opt.IDNumber = server.DefaultOption.IDNumber
		if opt.CoderType == "" {
			opt.CoderType = server.DefaultOption.CoderType
		}
		return opt, nil
	}
}

// Dial connects to an RPC server at the specified network address
func Connection(protocol, serverAddress string, options ...*server.Option) (client *Client, err error) {
	opt, err := parseOptions(options...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(protocol, serverAddress)
	if err != nil {
		return nil, err
	}
	// close the connection if client is nil
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return CreateClient(conn, opt)
}

func (client *Client) sendCall(call *Call) {
	// make sure that the client will send a complete request
	defer client.sending.Unlock()
	client.sending.Lock()

	// register this call.
	seqNumber, err := client.addCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// prepare request header
	client.header.ServiceMethod = call.ServiceMethod
	client.header.SequenceNumber = seqNumber
	client.header.Error = ""
	Error := client.message.Write(&client.header, call.Args)
	// encode and send the request
	if Error != nil {
		// call may be nil, it usually means that Write partially failed,
		// client has received the response and handled
		if call := client.deleteCall(seqNumber); call != nil {
			call.Error = Error
			call.done()
		}
	}
}

// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation.
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	switch{
	case done == nil:
		done = make(chan *Call, 10)
	case cap(done) == 0:
		log.Panic("rpc client: done channel is unbuffered")
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

// Call invokes the named function, waits for it to complete,
// and returns its error status.
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}