package client

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type LoadBalancingMode int

const (
	// RandomSelectMode use random generator to randomly select a server from the list
	RandomSelectMode LoadBalancingMode = iota
	// RoundRobinSelectMode use round-robbin algorithm to select the least used server from the list
	RoundRobinSelectMode
)

type Discovery interface {
	GetServer(loadBalancingMode LoadBalancingMode) (string, error)
	GetServerList() ([]string, error)
	RefreshServerList() error
	UpdateServerList(serverList []string) error
}

type DiscoveryMultiServers struct {
	randomNumber   *rand.Rand   // random number generated from math.Rand
	readWriteMutex sync.RWMutex // mutex that protect when read and write, which protect the following
	serverList     []string     // list of available servers
	serverIndex    int          // the recorded index for selected server, use for round-robin algorithm
}

var _ Discovery = (*DiscoveryMultiServers)(nil)

// CreateDiscoveryMultiServer creates a DiscoveryMultiServers instance
func CreateDiscoveryMultiServer(serverList []string) *DiscoveryMultiServers {
	discovery := &DiscoveryMultiServers{
		serverList:   serverList,
		randomNumber: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	discovery.serverIndex = discovery.randomNumber.Intn(math.MaxInt32 - 1)
	return discovery
}

// GetServer get an available server based on the load balancing loadBalancingMode
func (discovery *DiscoveryMultiServers) GetServer(loadBalancingMode LoadBalancingMode) (string, error) {
	discovery.readWriteMutex.Lock()
	defer discovery.readWriteMutex.Unlock()
	length := len(discovery.serverList)
	if length == 0 {
		return "", errors.New("RPC discovery -> GetServer: length of the serverList is 0, no available servers")
	}
	switch loadBalancingMode {
	case RandomSelectMode:
		return discovery.serverList[discovery.randomNumber.Intn(length)], nil
	case RoundRobinSelectMode:
		server := discovery.serverList[discovery.serverIndex%length] // servers could be updated, so modular length to ensure safety
		discovery.serverIndex = (discovery.serverIndex + 1) % length
		return server, nil
	default:
		return "", errors.New("RPC discovery -> GetServer: unrecognized loadBalancingMode")
	}
}

// GetServerList get all available servers in discovery as list
func (discovery *DiscoveryMultiServers) GetServerList() ([]string, error) {
	discovery.readWriteMutex.RLock()
	defer discovery.readWriteMutex.RUnlock()
	// return a copy of discovery.serverList
	serverList := make([]string, len(discovery.serverList), len(discovery.serverList))
	copy(serverList, discovery.serverList)
	return serverList, nil
}

// RefreshServerList is not applicable for MultiServersDiscovery, so return nil
func (discovery *DiscoveryMultiServers) RefreshServerList() error {
	
	return nil
}

// UpdateServerList update the servers of discovery dynamically
func (discovery *DiscoveryMultiServers) UpdateServerList(serverList []string) error {
	discovery.readWriteMutex.Lock()
	defer discovery.readWriteMutex.Unlock()
	discovery.serverList = serverList
	return nil
}
