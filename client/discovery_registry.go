package client

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type DiscoveryRegistryDiscovery struct {
	*DiscoveryMultiServers
	registry   string
	timeout    time.Duration
	lastUpdate time.Time
}

const defaultTimeout = time.Second * 10

func (registry *DiscoveryRegistryDiscovery) UpdateRegistry(servers []string) error {
	registry.readWriteMutex.Lock()
	defer registry.readWriteMutex.Unlock()
	registry.serverList = servers
	registry.lastUpdate = time.Now()
	return nil
}

func (registry *DiscoveryRegistryDiscovery) RefreshRegistry() error {
	registry.readWriteMutex.Lock()
	defer registry.readWriteMutex.Unlock()
	if registry.lastUpdate.Add(registry.timeout).After(time.Now()) {
		return nil
	}
	log.Println("Distributed RPC Registry: refresh servers from registry", registry.registry)
	response, Error := http.Get(registry.registry)
	if Error != nil {
		log.Println("Distributed RPC Registry refresh err:", Error)
		return Error
	}
	servers := strings.Split(response.Header.Get("X-Geerpc-Servers"), ",")
	registry.serverList = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			registry.serverList = append(registry.serverList, strings.TrimSpace(server))
		}
	}
	registry.lastUpdate = time.Now()
	return nil
}

func (registry *DiscoveryRegistryDiscovery) GetServer(mode LoadBalancingMode) (string, error) {
	if Error := registry.RefreshRegistry(); Error != nil {
		return "", Error
	}
	return registry.DiscoveryMultiServers.GetServer(mode)
}

func (registry *DiscoveryRegistryDiscovery) GetAllServers() ([]string, error) {
	if Error := registry.RefreshRegistry(); Error != nil {
		return nil, Error
	}
	return registry.DiscoveryMultiServers.GetServerList()
}

func NewDiscoveryRegistry(registerAddress string, timeout time.Duration) *DiscoveryRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultTimeout
	}
	discoveryRegistry := &DiscoveryRegistryDiscovery{
		DiscoveryMultiServers: CreateDiscoveryMultiServer(make([]string, 0)),
		registry:              registerAddress,
		timeout:               timeout,
	}
	return discoveryRegistry
}
