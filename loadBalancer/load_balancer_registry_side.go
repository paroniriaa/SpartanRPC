package loadBalancer

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type LoadBalancerRegistrySide struct {
	*LoadBalancerClientSide
	registryURL string
	timeout     time.Duration
	lastUpdate  time.Time
}

const defaultUpdateTimeout = time.Second * 10

func CreateLoadBalancerRegistrySide(registerURL string, timeout time.Duration) *LoadBalancerRegistrySide {
	log.Println("RPC Load Balancer(Registry Side) -> CreateLoadBalancerRegistrySide: creating RPC load balancer(registryURL side)...")
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	loadBalancerRegistrySide := &LoadBalancerRegistrySide{
		LoadBalancerClientSide: CreateLoadBalancerClientSide(make([]string, 0)),
		registryURL:            registerURL,
		timeout:                timeout,
	}
	return loadBalancerRegistrySide
}

func (loadBalancerRegistrySide *LoadBalancerRegistrySide) UpdateServerList(servers []string) error {
	//log.Println("RPC Load Balancer(Registry Side) -> UpdateServerList: RPC load balancer(registryURL side) manually update its maintained RPC server instance list...")
	loadBalancerRegistrySide.readWriteMutex.Lock()
	defer loadBalancerRegistrySide.readWriteMutex.Unlock()
	loadBalancerRegistrySide.serverList = servers
	loadBalancerRegistrySide.lastUpdate = time.Now()
	return nil
}

func (loadBalancerRegistrySide *LoadBalancerRegistrySide) RefreshServerList() error {
	//log.Println("RPC Load Balancer(Registry Side) -> RefreshServerList: RPC load balancer(registryURL side) automatically update its maintained RPC server instance list...")
	loadBalancerRegistrySide.readWriteMutex.Lock()
	defer loadBalancerRegistrySide.readWriteMutex.Unlock()
	if loadBalancerRegistrySide.lastUpdate.Add(loadBalancerRegistrySide.timeout).After(time.Now()) {
		return nil
	}
	//log.Println("RPC Load Balancer(Registry Side) -> RefreshServerList: refreshing servers from RPC registryURL...", loadBalancerRegistrySide.registryURL)
	resp, err := http.Get(loadBalancerRegistrySide.registryURL)
	if err != nil {
		log.Println("RPC Load Balancer(Registry Side) -> RefreshServerList error: refresh err: ", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("Get-SpartanRPC-AliveServers"), ",")
	loadBalancerRegistrySide.serverList = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			loadBalancerRegistrySide.serverList = append(loadBalancerRegistrySide.serverList, strings.TrimSpace(server))
		}
	}
	loadBalancerRegistrySide.lastUpdate = time.Now()
	return nil
}

func (loadBalancerRegistrySide *LoadBalancerRegistrySide) GetServer(loadBalancingMode LoadBalancingMode) (string, error) {
	//log.Println("RPC Load Balancer(Registry Side) -> GetServer: RPC load balancer(registryURL side) choose and return a RPC server instance based on load balancing mode...")
	if err := loadBalancerRegistrySide.RefreshServerList(); err != nil {
		return "", err
	}
	return loadBalancerRegistrySide.LoadBalancerClientSide.GetServer(loadBalancingMode)
}

func (loadBalancerRegistrySide *LoadBalancerRegistrySide) GetServerList() ([]string, error) {
	//log.Println("RPC Load Balancer(Registry Side) -> GetServerList: RPC load balancer(registryURL side) return all RPC severs instance in its maintained RPC server instance list...")
	if err := loadBalancerRegistrySide.RefreshServerList(); err != nil {
		return nil, err
	}
	return loadBalancerRegistrySide.LoadBalancerClientSide.GetServerList()
}
