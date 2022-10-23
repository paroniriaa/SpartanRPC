package loadBalancer

type LoadBalancingMode int

const (
	// RandomSelectMode use random generator to randomly select a server from the list
	RandomSelectMode LoadBalancingMode = iota
	// RoundRobinSelectMode use round-robbin algorithm to select the least used server from the list
	RoundRobinSelectMode
	// WeightRoundRobinSelectMode is based on round-robin algorithm, with an extra weight assigned to
	//each server instance, where greater weights assigned to high-performance servers
	WeightRoundRobinSelectMode
	// ConsistentHashSelectMode use a hash value based on some request criteria and deliver the request
	//to the correct server based on the hash value
	ConsistentHashSelectMode
)

type LoadBalancer interface {
	GetServer(loadBalancingMode LoadBalancingMode) (string, error)
	GetServerList() ([]string, error)
	RefreshServerList() error
	UpdateServerList(serverList []string) error
}
