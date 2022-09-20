package client

import (
	"Distributed-RPC-Framework/server"
	"context"
	"io"
	"log"
	"reflect"
	"sync"
)

type DiscoveryClient struct {
	discovery                   Discovery
	loadBalancingMode           LoadBalancingMode
	connectionInfo              *server.ConnectionInfo
	mutex                       sync.Mutex // protect following
	rpcServerAddressToClientMap map[string]*Client
}

var _ io.Closer = (*DiscoveryClient)(nil)

func CreateDiscoveryClient(discovery Discovery, loadBalancingMode LoadBalancingMode, connectionInfo *server.ConnectionInfo) *DiscoveryClient {
	discoveryClient := &DiscoveryClient{
		discovery:                   discovery,
		loadBalancingMode:           loadBalancingMode,
		connectionInfo:              connectionInfo,
		rpcServerAddressToClientMap: make(map[string]*Client),
	}
	return discoveryClient
}

func (discoveryClient *DiscoveryClient) Close() error {
	discoveryClient.mutex.Lock()
	defer discoveryClient.mutex.Unlock()
	for rpcServerAddress, client := range discoveryClient.rpcServerAddressToClientMap {
		_ = client.Close()
		delete(discoveryClient.rpcServerAddressToClientMap, rpcServerAddress)
	}
	return nil
}

func (discoveryClient *DiscoveryClient) dial(rpcServerAddress string) (*Client, error) {
	discoveryClient.mutex.Lock()
	defer discoveryClient.mutex.Unlock()
	client, keyExists := discoveryClient.rpcServerAddressToClientMap[rpcServerAddress]
	if keyExists && !client.IsAvailable() {
		_ = client.Close()
		delete(discoveryClient.rpcServerAddressToClientMap, rpcServerAddress)
		client = nil
	}
	if client == nil {
		var Error error
		client, Error = XMakeDial(rpcServerAddress, discoveryClient.connectionInfo)
		if Error != nil {
			return nil, Error
		}
		discoveryClient.rpcServerAddressToClientMap[rpcServerAddress] = client
	}
	return client, nil
}

func (discoveryClient *DiscoveryClient) call(rpcServerAddress string, contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	client, Error := discoveryClient.dial(rpcServerAddress)
	if Error != nil {
		return Error
	}
	log.Printf("RPC discoveryClient -> Call: Client %p invoking RPC request to RPC server %s on function %s with inputs -> %v", client, rpcServerAddress, serviceDotMethod, inputs)
	return client.Call(serviceDotMethod, inputs, output, contextInfo)
}

func (discoveryClient *DiscoveryClient) Call(contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	rpcServerAddress, err := discoveryClient.discovery.GetServer(discoveryClient.loadBalancingMode)
	if err != nil {
		return err
	}
	return discoveryClient.call(rpcServerAddress, contextInfo, serviceDotMethod, inputs, output)
}

func (discoveryClient *DiscoveryClient) Broadcast(contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	serverList, Error := discoveryClient.discovery.GetServerList()
	if Error != nil {
		return Error
	}
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex // protect broadcastError and replyDone
	var broadcastError error
	replyDone := output == nil // if output is nil, no need to set value
	contextInfo, cancelContext := context.WithCancel(contextInfo)
	for _, rpcServerAddress := range serverList {
		waitGroup.Add(1)
		go func(address string) {
			defer waitGroup.Done()
			var clonedReply interface{}
			if output != nil {
				clonedReply = reflect.New(reflect.ValueOf(output).Elem().Type()).Interface()
			}
			err := discoveryClient.call(address, contextInfo, serviceDotMethod, inputs, clonedReply)
			mutex.Lock()
			if err != nil && broadcastError == nil {
				//if err != nil {
				broadcastError = err
				cancelContext() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				log.Printf("RPC DiscoveryClient -> Broadcast: DiscoveryClient %p recieved RPC response from RPC server %s first for RPC request invoking function %s with inputs -> %v", discoveryClient, address, serviceDotMethod, inputs)
				reflect.ValueOf(output).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mutex.Unlock()
		}(rpcServerAddress)
	}
	waitGroup.Wait()
	return broadcastError
}
