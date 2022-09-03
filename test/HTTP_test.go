package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"testing"
)

func TestXDial(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	t.Run("WindowsXDial", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			addressChannel := make(chan struct{})
			address := "localhost:8005"
			go func() {
				_ = os.Remove(address)
				listener, err := net.Listen("tcp", address)
				if err != nil {
					t.Error("failed to listen windows socket")
					return
				}
				server.RegisterHandlerHTTP()
				addressChannel <- struct{}{}
				_ = http.Serve(listener, nil)
				//server.AcceptConnection(listener)
			}()
			<-addressChannel
			_, err := client.XMakeDial("http@" + address)
			_assert(err == nil, "failed to connect windows socket")
		} else {
			log.Println("current GO OS is not windows, corresponding sub tests has been dumped")
		}
	})

	t.Run("LinuxXDial", func(t *testing.T) {
		if runtime.GOOS == "linux" {
			ch := make(chan struct{})
			addr := "/tmp/sRPC.sock"
			go func() {
				_ = os.Remove(addr)
				l, err := net.Listen("unix", addr)
				if err != nil {
					t.Error("failed to listen unix socket")
					return
				}
				ch <- struct{}{}
				server.AcceptConnection(l)
			}()
			<-ch
			_, err := client.XMakeDial("unix@" + addr)
			_assert(err == nil, "failed to connect unix socket")
		} else {
			log.Println("current GO OS is not linux, corresponding sub tests has been dumped")
		}
	})
}
