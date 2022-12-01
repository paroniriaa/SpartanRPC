package loadBalancer

import (
	"errors"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

type LoadBalancerClientSide struct {
	randomNumber   *rand.Rand   // random number generated from math.Rand
	readWriteMutex sync.RWMutex // mutex that protect when read and write, which protect the following
	serverList     []string     // list of available servers
	serverIndex    int          // the recorded index for selected server, use for round-robin algorithm
}

var _ LoadBalancer = (*LoadBalancerClientSide)(nil)

// CreateLoadBalancerClientSide creates a LoadBalancerClientSide instance
func CreateLoadBalancerClientSide(serverList []string) *LoadBalancerClientSide {
	//log.Println("RPC Load Balancer Client Side: creating client side load balancer...")
	loadBalancerClientSide := &LoadBalancerClientSide{
		serverList:   serverList,
		randomNumber: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	loadBalancerClientSide.serverIndex = loadBalancerClientSide.randomNumber.Intn(math.MaxInt32 - 1)
	log.Printf("RPC Load Balancer(Client Side) -> CreateLoadBalancerClientSide: created RPC load balancer(Client side) %p with field -> %+v...", loadBalancerClientSide, loadBalancerClientSide)
	return loadBalancerClientSide
}

// GetServer get an available server based on the load balancing loadBalancingMode
func (loadBalancerClientSide *LoadBalancerClientSide) GetServer(loadBalancingMode LoadBalancingMode) (string, error) {
	loadBalancerClientSide.readWriteMutex.Lock()
	defer loadBalancerClientSide.readWriteMutex.Unlock()
	log.Printf("RPC Load Balancer(Client Side) -> GetServer: RPC load balancer(client side) %p choose and return an RPC server instance based on load balancing mode...", loadBalancerClientSide)
	length := len(loadBalancerClientSide.serverList)
	if length == 0 {
		return "", errors.New("RPC Load Balancer(Client Side) -> GetServer: length of the serverList is 0, no available servers")
	}
	switch loadBalancingMode {
	case RandomSelectMode:
		return loadBalancerClientSide.serverList[loadBalancerClientSide.randomNumber.Intn(length)], nil
	case RoundRobinSelectMode:
		// servers could be updated, so modular length to ensure safety
		server := loadBalancerClientSide.serverList[loadBalancerClientSide.serverIndex%length]
		loadBalancerClientSide.serverIndex = (loadBalancerClientSide.serverIndex + 1) % length
		return server, nil
	default:
		return "", errors.New("RPC Load Balancer(Client Side) -> GetServer: unrecognized loadBalancingMode")
	}
}

// GetServerList get all available servers in loadBalancer as list
func (loadBalancerClientSide *LoadBalancerClientSide) GetServerList() ([]string, error) {
	loadBalancerClientSide.readWriteMutex.RLock()
	defer loadBalancerClientSide.readWriteMutex.RUnlock()
	log.Printf("RPC Load Balancer(Client Side) -> GetServerList: RPC load balancer(cleint side) %p return all RPC severs instance in its maintained RPC server instance list...", loadBalancerClientSide)
	// return a copy of loadBalancerClientSide.serverList
	serverList := make([]string, len(loadBalancerClientSide.serverList), len(loadBalancerClientSide.serverList))
	copy(serverList, loadBalancerClientSide.serverList)
	return serverList, nil
}

// RefreshServerList is not applicable for client-side load balancer, so return nil
func (loadBalancerClientSide *LoadBalancerClientSide) RefreshServerList() error {
	return nil
}

// UpdateServerList update the servers of loadBalancer dynamically
func (loadBalancerClientSide *LoadBalancerClientSide) UpdateServerList(serverList []string) error {
	loadBalancerClientSide.readWriteMutex.Lock()
	defer loadBalancerClientSide.readWriteMutex.Unlock()
	log.Printf("RPC Load Balancer(Client Side) -> UpdateServerList: RPC load balancer(client side) %p manually update its maintained RPC server instance list...", loadBalancerClientSide)
	loadBalancerClientSide.serverList = serverList
	return nil
}
