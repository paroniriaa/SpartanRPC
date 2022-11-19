package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Registry is an RPC register center that provides the following functions.
// add an RPC server and receive its heartbeat to keep it alive in its server list.
// returns all alive RPC servers in the rpcServerAddressToServerInfoMap and delete dead RPC servers simultaneously.
type Registry struct {
	RegistryAddress                 string
	timeout                         time.Duration
	mutex                           sync.Mutex // protect following
	rpcServerAddressToServerInfoMap map[string]*ServerInfo
}

type ServerInfo struct {
	ServerAddress  string
	lastUpdateTime time.Time
}

// default timeout value set to be 5 minutes,
// indicating that any server has lastUpdateTime + DefaultTimeout <= now
// will be treated as unavailable server and will be removed
const (
	defaultPath    = "/_srpc_/registry"
	DefaultAddress = "http://localhost:9999/_srpc_/registry"
	DefaultPort    = ":9999"
	DefaultTimeout = time.Minute * 5
)

// CreateRegistry create a registry instance with timeout setting
func CreateRegistry(registryAddress string, timeout time.Duration) *Registry {
	log.Println("RPC Registry -> CreateRegistry: creating RPC registry...")
	return &Registry{
		RegistryAddress:                 registryAddress,
		rpcServerAddressToServerInfoMap: make(map[string]*ServerInfo),
		timeout:                         timeout,
	}
}

func (registry *Registry) registerServer(addr string) {
	log.Println("RPC Registry -> registerServer: RPC registry registering server instance...")
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	serverInfo := registry.rpcServerAddressToServerInfoMap[addr]
	if serverInfo == nil {
		registry.rpcServerAddressToServerInfoMap[addr] = &ServerInfo{ServerAddress: addr, lastUpdateTime: time.Now()}
	} else {
		// if the server already exists, update its lastUpdateTime time as now to keep it alive
		serverInfo.lastUpdateTime = time.Now()
	}
}

func (registry *Registry) getAliveServerList() []string {
	log.Println("RPC Registry -> getAliveServerList: RPC registry return list of aliveServerList server instance (and delete any expire server instances)...")
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	var aliveServerList []string
	for serverAddress, serverInfo := range registry.rpcServerAddressToServerInfoMap {
		// if the current registry instance set the timeout value to 0, we treat all servers as always alive(not recommended)
		// if the current server's lastUpdateTime + lastUpdateTime + current registry's timeout value > now
		// then this server will be treated as an alive server
		if registry.timeout == 0 || serverInfo.lastUpdateTime.Add(registry.timeout).After(time.Now()) {
			aliveServerList = append(aliveServerList, serverAddress)
		} else {
			delete(registry.rpcServerAddressToServerInfoMap, serverAddress)
		}
	}
	sort.Strings(aliveServerList)
	return aliveServerList
}

// ServeHTTP is the registry HTTP endpoint that runs at /_srpc_/registry
func (registry *Registry) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.Method {
	//GET returns all the alive servers instance currently available in the registry, used "SpartanRPC-AliveServers" as customized HTTP header field
	case "GET":
		log.Println("RPC Registry -> ServeHTTP: RPC registry serving HTTP GET request...")
		// keep it simple, server is in request.Header
		responseWriter.Header().Set("SpartanRPC-AliveServers", strings.Join(registry.getAliveServerList(), ","))
		//POST register/send heartbeat for the server instance to the registry, used "SpartanRPC-AliveServer" as customized HTTP header field
	case "POST":
		log.Println("RPC Registry -> ServeHTTP: RPC registry serving HTTP POST request...")
		// keep it simple, server is in request.Header
		serverAddress := request.Header.Get("SpartanRPC-AliveServer")
		if serverAddress == "" {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		registry.registerServer(serverAddress)
	default:
		responseWriter.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for Registry messages on the registryPath
func (registry *Registry) HandleHTTP() {
	log.Println("RPC Registry -> HandleHTTP: RPC registry initializing an HTTP handler for message receiving/sending...")
	http.Handle(defaultPath, registry)
	log.Println("RPC Registry -> HandleHTTP: RPC registry path:", defaultPath, "")
}

/*
//var DefaultRegister = CreateRegistry(DefaultAddress, DefaultTimeout)

func HandleHTTP() {
	DefaultRegister.HandleHTTP(defaultPath)
}
*/
