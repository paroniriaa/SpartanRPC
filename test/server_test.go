package test

import (
	"Distributed-RPC-Framework/coder"
	"Distributed-RPC-Framework/server"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

// TODO: struct
type Args struct {
	A, B int
}

type Reply struct {
	C int
}

type ArithAddResp struct {
	Id     any   `json:"id"`
	Result Reply `json:"result"`
	Error  any   `json:"error"`
}

type Arith int


func startServer(address chan string) {
	portNumber, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal("network issue:", err)
	}
	log.Println("startServer RPC server on port", portNumber.Addr())
	address <- portNumber.Addr().String()
	server.AcceptConnection(portNumber)
}
/*
func TestServer(test *testing.T) {
	test.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	address := make(chan string)
	go startServer(address)

	connection, _ := net.Dial("tcp", <-address)
	defer func() { _ = connection.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(connection).Encode(server.DefaultConnectionInfo)
	communication := coder.NewJsonCoder(connection)

	header := &coder.Header{
		ServiceDotMethod: "Test.Echo",
		SequenceNumber:   uint64(1),
	}
	request := "RPC Sequence Number " + strconv.Itoa(1)
	log.Println("Request:", request)
	_ = communication.EncodeMessageHeaderAndBody(header, request)
	_ = communication.DecodeMessageHeader(header)
	var response string
	_ = communication.DecodeMessageBody(&response)
	log.Println("Response:", response)

}
*/

func TestServer(test *testing.T) {
	test.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	addr := make(chan string)
	go startServer(addr)

	// in fact, following code is like a simple geerpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	// send options
	_ = json.NewEncoder(conn).Encode(server.DefaultConnectionInfo)
	cc := coder.NewJsonCoder(conn)
	// send request & receive response
	for i := 0; i < 5; i++ {
		h := &coder.Header{
			ServiceDotMethod: "Foo.Sum",
			SequenceNumber:   uint64(i),
		}
		_ = cc.EncodeMessageHeaderAndBody(h, fmt.Sprintf("geerpc req %d", h.SequenceNumber))
		_ = cc.DecodeMessageHeader(h)
		var reply string
		_ = cc.DecodeMessageBody(&reply)
		log.Println("reply:", reply)
	}
}


/*
func TestServerNoParams(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	address := make(chan string)
	go createServer(srv.LocalAddr())
	dec := coder.Coder(cli)

	fmt.Fprintf(cli, `{"method": "Arith.Add", "id": "123"}`)
	var resp ArithAddResp
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("Decode after no params: %s", err)
	}
	if resp.Error == nil {
		t.Fatalf("Expected error, got nil")
	}
}


func TestServerEmptyMessage(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go ServeConn(srv)
	dec := json.NewDecoder(cli)

	fmt.Fprintf(cli, "{}")
	var resp ArithAddResp
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("Decode after empty: %s", err)
	}
	if resp.Error == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestServerErrorHasNullResult(t *testing.T) {
	var out bytes.Buffer
	sc := NewServerCodec(struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		Reader: strings.NewReader(`{"method": "Arith.Add", "id": "123", "params": []}`),
		Writer: &out,
		Closer: io.NopCloser(nil),
	})
	r := new(rpc.Request)
	if err := sc.ReadRequestHeader(r); err != nil {
		t.Fatal(err)
	}
	const valueText = "the value we don't want to see"
	const errorText = "some error"
	err := sc.WriteResponse(&rpc.Response{
		ServiceMethod: "Method",
		Seq:           1,
		Error:         errorText,
	}, valueText)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), errorText) {
		t.Fatalf("Response didn't contain expected error %q: %s", errorText, &out)
	}
	if strings.Contains(out.String(), valueText) {
		t.Errorf("Response contains both an error and value: %s", &out)
	}
}
*/
