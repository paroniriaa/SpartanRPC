package client

import (
	"Distributed-RPC-Framework/server"
	"context"
	"io"
	"reflect"
	"sync"
)

type Discovery_Client struct {
	discovery      Discovery
	mode           LoadBalancingMode
	connectionInfo *server.ConnectionInfo
	mutex          sync.Mutex // protect following
	clients        map[string]*Client
}

var _ io.Closer = (*Discovery_Client)(nil)

func CreateDiscoveryClient(discovery Discovery, mode LoadBalancingMode, connectionInfo *server.ConnectionInfo) *Discovery_Client {
	return &Discovery_Client{discovery: discovery, mode: mode, connectionInfo: connectionInfo, clients: make(map[string]*Client)}
}

func (DiscoveryClient *Discovery_Client) Close() error {
	DiscoveryClient.mutex.Lock()
	defer DiscoveryClient.mutex.Unlock()
	for key, client := range DiscoveryClient.clients {
		_ = client.Close()
		delete(DiscoveryClient.clients, key)
	}
	return nil
}

func (DiscoveryClient *Discovery_Client) dial(address string) (*Client, error) {
	DiscoveryClient.mutex.Lock()
	defer DiscoveryClient.mutex.Unlock()
	client, ok := DiscoveryClient.clients[address]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(DiscoveryClient.clients, address)
		client = nil
	}
	if client == nil {
		var Error error
		client, Error = XMakeDial(address, DiscoveryClient.connectionInfo)
		if Error != nil {
			return nil, Error
		}
		DiscoveryClient.clients[address] = client
	}
	return client, nil
}

func (DiscoveryClient *Discovery_Client) call(address string, contextInfo context.Context, serviceMethod string, args, reply interface{}) error {
	client, Error := DiscoveryClient.dial(address)
	if Error != nil {
		return Error
	}
	return client.Call(serviceMethod, args, reply, contextInfo)
}

func (DiscoveryClient *Discovery_Client) Call(contextInfo context.Context, serviceMethod string, args, reply interface{}) error {
	address, err := DiscoveryClient.discovery.GetServer(DiscoveryClient.mode)
	if err != nil {
		return err
	}
	return DiscoveryClient.call(address, contextInfo, serviceMethod, args, reply)
}

func (DiscoveryClient *Discovery_Client) Broadcast(contextInfo context.Context, serviceMethod string, args, reply interface{}) error {
	servers, Error := DiscoveryClient.discovery.GetAllServers()
	if Error != nil {
		return Error
	}
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex // protect e and replyDone
	var e error
	replyDone := reply == nil // if reply is nil, don't need to set value
	contextInfo, cancel := context.WithCancel(contextInfo)
	for _, address := range servers {
		waitGroup.Add(1)
		go func(address string) {
			defer waitGroup.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			Error := DiscoveryClient.call(address, contextInfo, serviceMethod, args, clonedReply)
			mutex.Lock()
			if Error != nil && e == nil {
				e = Error
				cancel() // if any call failed, cancel unfinished calls
			}
			if Error == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mutex.Unlock()
		}(address)
	}
	waitGroup.Wait()
	return e
}
