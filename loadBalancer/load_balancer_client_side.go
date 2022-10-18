package loadBalancer

import (
	"errors"
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
	loadBalancerDiscovery := &LoadBalancerClientSide{
		serverList:   serverList,
		randomNumber: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	loadBalancerDiscovery.serverIndex = loadBalancerDiscovery.randomNumber.Intn(math.MaxInt32 - 1)
	return loadBalancerDiscovery
}

// GetServer get an available server based on the load balancing loadBalancingMode
func (loadBalancerClientSide *LoadBalancerClientSide) GetServer(loadBalancingMode LoadBalancingMode) (string, error) {
	loadBalancerClientSide.readWriteMutex.Lock()
	defer loadBalancerClientSide.readWriteMutex.Unlock()
	length := len(loadBalancerClientSide.serverList)
	if length == 0 {
		return "", errors.New("RPC loadBalancerClientSide -> GetServer: length of the serverList is 0, no available servers")
	}
	switch loadBalancingMode {
	case RandomSelectMode:
		return loadBalancerClientSide.serverList[loadBalancerClientSide.randomNumber.Intn(length)], nil
	case RoundRobinSelectMode:
		server := loadBalancerClientSide.serverList[loadBalancerClientSide.serverIndex%length] // servers could be updated, so modular length to ensure safety
		loadBalancerClientSide.serverIndex = (loadBalancerClientSide.serverIndex + 1) % length
		return server, nil
	default:
		return "", errors.New("RPC loadBalancerClientSide -> GetServer: unrecognized loadBalancingMode")
	}
}

// GetServerList get all available servers in loadBalancer as list
func (loadBalancerClientSide *LoadBalancerClientSide) GetServerList() ([]string, error) {
	loadBalancerClientSide.readWriteMutex.RLock()
	defer loadBalancerClientSide.readWriteMutex.RUnlock()
	// return a copy of loadBalancerClientSide.serverList
	serverList := make([]string, len(loadBalancerClientSide.serverList), len(loadBalancerClientSide.serverList))
	copy(serverList, loadBalancerClientSide.serverList)
	return serverList, nil
}

// RefreshServerList is not applicable for MultiServersDiscovery, so return nil
func (loadBalancerClientSide *LoadBalancerClientSide) RefreshServerList() error {
	return nil
}

// UpdateServerList update the servers of loadBalancer dynamically
func (loadBalancerClientSide *LoadBalancerClientSide) UpdateServerList(serverList []string) error {
	loadBalancerClientSide.readWriteMutex.Lock()
	defer loadBalancerClientSide.readWriteMutex.Unlock()
	loadBalancerClientSide.serverList = serverList
	return nil
}
