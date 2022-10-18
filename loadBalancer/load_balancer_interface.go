package loadBalancer

type LoadBalancingMode int

const (
	// RandomSelectMode use random generator to randomly select a server from the list
	RandomSelectMode LoadBalancingMode = iota
	// RoundRobinSelectMode use round-robbin algorithm to select the least used server from the list
	RoundRobinSelectMode
)

type LoadBalancer interface {
	GetServer(loadBalancingMode LoadBalancingMode) (string, error)
	GetServerList() ([]string, error)
	RefreshServerList() error
	UpdateServerList(serverList []string) error
}
