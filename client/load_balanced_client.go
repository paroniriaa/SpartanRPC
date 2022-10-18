package client

import (
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/server"
	"context"
	"io"
	"log"
	"reflect"
	"sync"
)

type LoadBalancedClient struct {
	loadBalancer             loadBalancer.LoadBalancer
	loadBalancingMode        loadBalancer.LoadBalancingMode
	connectionInfo           *server.ConnectionInfo
	mutex                    sync.Mutex // protect following
	serverAddressToClientMap map[string]*Client
}

var _ io.Closer = (*LoadBalancedClient)(nil)

func CreateLoadBalancedClient(loadBalancer loadBalancer.LoadBalancer, loadBalancingMode loadBalancer.LoadBalancingMode, connectionInfo *server.ConnectionInfo) *LoadBalancedClient {
	loadBalancedClient := &LoadBalancedClient{
		loadBalancer:             loadBalancer,
		loadBalancingMode:        loadBalancingMode,
		connectionInfo:           connectionInfo,
		serverAddressToClientMap: make(map[string]*Client),
	}
	return loadBalancedClient
}

func (loadBalancedClient *LoadBalancedClient) Close() error {
	loadBalancedClient.mutex.Lock()
	defer loadBalancedClient.mutex.Unlock()
	for rpcServerAddress, client := range loadBalancedClient.serverAddressToClientMap {
		_ = client.Close()
		delete(loadBalancedClient.serverAddressToClientMap, rpcServerAddress)
	}
	return nil
}

func (loadBalancedClient *LoadBalancedClient) dial(rpcServerAddress string) (*Client, error) {
	loadBalancedClient.mutex.Lock()
	defer loadBalancedClient.mutex.Unlock()
	client, keyExists := loadBalancedClient.serverAddressToClientMap[rpcServerAddress]
	if keyExists && !client.IsAvailable() {
		_ = client.Close()
		delete(loadBalancedClient.serverAddressToClientMap, rpcServerAddress)
		client = nil
	}
	if client == nil {
		var Error error
		client, Error = XMakeDial(rpcServerAddress, loadBalancedClient.connectionInfo)
		if Error != nil {
			return nil, Error
		}
		loadBalancedClient.serverAddressToClientMap[rpcServerAddress] = client
	}
	return client, nil
}

func (loadBalancedClient *LoadBalancedClient) call(rpcServerAddress string, contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	client, Error := loadBalancedClient.dial(rpcServerAddress)
	if Error != nil {
		return Error
	}
	log.Printf("RPC loadBalancedClient -> Call: RPC client %p invoking RPC request to RPC server %s on function %s with inputs -> %v", client, rpcServerAddress, serviceDotMethod, inputs)
	return client.Call(serviceDotMethod, inputs, output, contextInfo)
}

func (loadBalancedClient *LoadBalancedClient) Call(contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	rpcServerAddress, err := loadBalancedClient.loadBalancer.GetServer(loadBalancedClient.loadBalancingMode)
	if err != nil {
		return err
	}
	return loadBalancedClient.call(rpcServerAddress, contextInfo, serviceDotMethod, inputs, output)
}

func (loadBalancedClient *LoadBalancedClient) BroadcastCall(contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	serverList, Error := loadBalancedClient.loadBalancer.GetServerList()
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
			err := loadBalancedClient.call(address, contextInfo, serviceDotMethod, inputs, clonedReply)
			mutex.Lock()
			if err != nil && broadcastError == nil {
				//if err != nil {
				broadcastError = err
				cancelContext() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				log.Printf("RPC LoadBalancedClient -> BroadcastCall: LoadBalancedClient %p recieved RPC response from RPC server %s first for RPC request invoking function %s with inputs -> %v", loadBalancedClient, address, serviceDotMethod, inputs)
				reflect.ValueOf(output).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mutex.Unlock()
		}(rpcServerAddress)
	}
	waitGroup.Wait()
	return broadcastError
}
