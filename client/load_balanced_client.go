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
	log.Printf("RPC loadBalancedClient -> CreateLoadBalancedClient: RPC loadBalancedClient %p created with field -> %+v", loadBalancedClient, loadBalancedClient)
	return loadBalancedClient
}

func (loadBalancedClient *LoadBalancedClient) Close() error {
	loadBalancedClient.mutex.Lock()
	defer loadBalancedClient.mutex.Unlock()
	for rpcServerAddress, client := range loadBalancedClient.serverAddressToClientMap {
		_ = client.Close()
		delete(loadBalancedClient.serverAddressToClientMap, rpcServerAddress)
	}
	log.Printf("RPC loadBalancedClient -> Close: All managed RPC clients in RPC loadBalancedClient %p is closed", loadBalancedClient)
	return nil
}

func (loadBalancedClient *LoadBalancedClient) dial(rpcServerAddress string) (*Client, error) {
	loadBalancedClient.mutex.Lock()
	defer loadBalancedClient.mutex.Unlock()
	client, keyExists := loadBalancedClient.serverAddressToClientMap[rpcServerAddress]
	if keyExists && !client.IsAvailable() {
		_ = client.Close()
		log.Printf("RPC loadBalancedClient -> dial: fetched RPC client %p is not availavle, set RPC client to nil and deleted map entry for RPC server %s", client, rpcServerAddress)
		delete(loadBalancedClient.serverAddressToClientMap, rpcServerAddress)
		client = nil
	}
	if client == nil {
		var Error error
		log.Printf("RPC loadBalancedClient -> dial: fetched RPC client is nil, initializing a new RPC client with process XMakeDial()...")
		client, Error = XMakeDial(rpcServerAddress, loadBalancedClient.connectionInfo)
		if Error != nil {
			return nil, Error
		}
		loadBalancedClient.serverAddressToClientMap[rpcServerAddress] = client
	}
	return client, nil
}

func (loadBalancedClient *LoadBalancedClient) call(rpcServerAddress string, contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	log.Printf("RPC loadBalancedClient -> call: initializing procdss dial() to get RPC client that dialed and connected to RPC server %s...", rpcServerAddress)
	client, Error := loadBalancedClient.dial(rpcServerAddress)
	if Error != nil {
		return Error
	}
	log.Printf("RPC loadBalancedClient -> call: RPC loadBalancedClient %p initializaing RPC client %p process Call() on RPC server %s...", loadBalancedClient, client, rpcServerAddress)
	return client.Call(serviceDotMethod, inputs, output, contextInfo)
}

func (loadBalancedClient *LoadBalancedClient) Call(contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	log.Printf("RPC loadBalancedClient -> Call: RPC loadBalancedClient %p initializaing process GetServer() to get RPC server based on load balancing mode %v...", loadBalancedClient, loadBalancedClient.loadBalancingMode)
	rpcServerAddress, err := loadBalancedClient.loadBalancer.GetServer(loadBalancedClient.loadBalancingMode)
	if err != nil {
		return err
	}
	return loadBalancedClient.call(rpcServerAddress, contextInfo, serviceDotMethod, inputs, output)
}

func (loadBalancedClient *LoadBalancedClient) BroadcastCall(contextInfo context.Context, serviceDotMethod string, inputs, output interface{}) error {
	log.Printf("RPC loadBalancedClient -> BroadcastCall: RPC loadBalancedClient %p initializaing process GetServerList() to get all RPC servers...", loadBalancedClient)
	serverList, getServerErr := loadBalancedClient.loadBalancer.GetServerList()
	if getServerErr != nil {
		return getServerErr
	}
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex // protect broadcastError and replyDone
	var broadcastError error
	replyDone := output == nil // if output is nil, no need to set value
	contextInfo, cancelContext := context.WithCancel(contextInfo)
	log.Printf("RPC loadBalancedClient -> BroadcastCall: RPC loadBalancedClient %p initializaing process call() for all RPC servers...", loadBalancedClient)
	for _, rpcServerAddress := range serverList {
		waitGroup.Add(1)
		go func(serverAddress string) {
			defer waitGroup.Done()
			var clonedReply interface{}
			if output != nil {
				clonedReply = reflect.New(reflect.ValueOf(output).Elem().Type()).Interface()
			}
			err := loadBalancedClient.call(serverAddress, contextInfo, serviceDotMethod, inputs, clonedReply)
			mutex.Lock()
			if err != nil && broadcastError == nil {
				//if getServerErr != nil {
				broadcastError = err
				cancelContext() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				client, _ := loadBalancedClient.serverAddressToClientMap[serverAddress]
				log.Printf("RPC loadBalancedClient -> BroadcastCall: RPC client %s recieved RPC response from RPC server %s first for RPC request invoking function %s with inputs -> %v", client.clientAddress, serverAddress, serviceDotMethod, inputs)
				//Set the output from the responses RPC call
				reflect.ValueOf(output).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mutex.Unlock()
		}(rpcServerAddress)
	}
	waitGroup.Wait()
	cancelContext()
	if broadcastError != nil {
		log.Printf("RPC loadBalancedClient -> BroadcastCall error: RPC LoadBalancedClient %p has one RPC Call failed due to %s, all unfinsihed RPC call has been cancled...", loadBalancedClient, broadcastError)
	}
	return broadcastError
}
