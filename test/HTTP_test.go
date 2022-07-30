package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"net"
	"os"
	"runtime"
	"testing"
)

func TestXDial(t *testing.T) {
	if runtime.GOOS == "window" {
		ch := make(chan struct{})
		addr := "/tmp/RPC.sock"
		go func() {
			_ = os.Remove(addr)
			l, err := net.Listen("winsock", addr)
			if err != nil {
				t.Fatal("failed to listen windows socket")
			}
			ch <- struct{}{}
			server.AcceptConnection(l)
		}()
		<-ch
		_, err := client.XMakeDial("widnows@" + addr)
		_assert(err == nil, "failed to connect windows socket")
	}
}