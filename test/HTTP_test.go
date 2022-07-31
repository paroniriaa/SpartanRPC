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

	t.Run("windows", func(t *testing.T) {
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
	})


	t.Run("linux", func(t *testing.T) {
		if runtime.GOOS == "linux" {
			ch := make(chan struct{})
			addr := "/tmp/RPC.sock"
			go func() {
				_ = os.Remove(addr)
				l, err := net.Listen("unix", addr)
				if err != nil {
					t.Fatal("failed to listen unix socket")
				}
				ch <- struct{}{}
				server.AcceptConnection(l)
			}()
			<-ch
			_, err := client.XMakeDial("linux@" + addr)
			_assert(err == nil, "failed to connect unix socket")
		}
	})
}