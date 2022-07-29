package test

import (
	"Distributed-RPC-Framework/client"
	"Distributed-RPC-Framework/server"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClient_dialTimeout(t *testing.T) {
	t.Parallel()
	l, _ := net.Listen("tcp", ":0")

	f := func(conn net.Conn, opt *server.ConnectionInfo) (client *client.Client, err error) {
		_ = conn.Close()
		time.Sleep(time.Second * 2)
		return nil, nil
	}
	t.Run("timeout", func(t *testing.T) {
		_, err := client.MakeDialWithTimeout(f, "tcp", l.Addr().String(), &server.ConnectionInfo{ConnectionTimeout: time.Second})
		_assert(err != nil && strings.Contains(err.Error(), "connect timeout"), "expect a timeout error")
	})
	t.Run("0", func(t *testing.T) {
		_, err := client.MakeDialWithTimeout(f, "tcp", l.Addr().String(), &server.ConnectionInfo{ConnectionTimeout: 0})
		_assert(err == nil, "0 means no limit")
	})
}