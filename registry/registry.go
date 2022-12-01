package registry

import (
	"errors"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Registry is an RPC register center that provides the following functions.
// add an RPC server and receive its heartbeat to keep it alive in its server list.
// returns all alive RPC servers in the RpcServerAddressToServerInfoMap and delete dead RPC servers simultaneously.
type Registry struct {
	Listener                        net.Listener
	RegistryAddress                 string
	RegistryURL                     string
	Timeout                         time.Duration
	mutex                           sync.Mutex // protect following
	RpcServerAddressToServerInfoMap map[string]*ServerInfo
}

type ServerInfo struct {
	ServerAddress  string
	lastUpdateTime time.Time
}

// default Timeout value set to be 5 minutes,
// indicating that any server has lastUpdateTime + DefaultTimeout <= now
// will be treated as unavailable server and will be removed
const (
	DefaultRegistryPath = "/_srpc_/registry"
	//DefaultTimeout should not be less than 1 minute for performance purpose
	DefaultTimeout = time.Minute * 2
)

// CreateRegistry create a registry instance with Timeout setting
func CreateRegistry(listener net.Listener, timeout time.Duration) (*Registry, error) {
	//log.Printf("RPC registry -> CreateRegistry: creating RPC registry on port %s...", listener.Addr().String())
	if listener == nil {
		return nil, errors.New("RPC server > CreateServer error: Network listener should not be nil, but received nil")
	}
	var registryURL string
	registryListenerAddress := listener.Addr().String()
	if registryListenerAddress[:4] == "[::]" {
		//registryListenerAddress -> "[::]:1234" -> port extraction needed
		registryURL = "http://localhost" + registryListenerAddress[4:] + DefaultRegistryPath
	} else {
		//registryListenerAddress -> "127.0.0.1:1234", port extraction not needed
		registryURL = "http://" + registryListenerAddress + DefaultRegistryPath
	}
	//http@registryAddress will hide the transportation protocol info, use listener.Addr().Network() to fetch the transportation protocol info when needed
	registryAddress := "http" + "@" + listener.Addr().String()
	//log.Printf("RPC registry -> CreateRegistry: created RPC registry on HTTP end-point %s...", registryURL)
	registry := &Registry{
		Listener:                        listener,
		RegistryAddress:                 registryAddress,
		RegistryURL:                     registryURL,
		RpcServerAddressToServerInfoMap: make(map[string]*ServerInfo),
		Timeout:                         timeout,
	}
	log.Printf("RPC registry -> CreateRegistry: created RPC registry %s with field -> %+v", registry.RegistryAddress, registry)
	return registry, nil
}

func (registry *Registry) registerServer(serverAddress string) {
	//log.Printf("RPC registry -> registerServer: RPC registry %p updating server instance %s...", registry, serverAddress)
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	serverInfo := registry.RpcServerAddressToServerInfoMap[serverAddress]
	if serverInfo == nil {
		registry.RpcServerAddressToServerInfoMap[serverAddress] = &ServerInfo{ServerAddress: serverAddress, lastUpdateTime: time.Now()}
	} else {
		// if the server already exists, update its lastUpdateTime time as now to keep it alive
		serverInfo.lastUpdateTime = time.Now()
	}
	log.Printf("RPC registry -> registerServer: RPC registry %s finished RPC server instance update with address %s, and the current alive server map is: %+v", registry.RegistryAddress, serverAddress, registry.RpcServerAddressToServerInfoMap)
}

func (registry *Registry) getAliveServerList() []string {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	var aliveServerList []string
	for serverAddress, serverInfo := range registry.RpcServerAddressToServerInfoMap {
		// if the current registry instance set the Timeout value to 0, we treat all servers as always alive(not recommended)
		// if the current server's lastUpdateTime + lastUpdateTime + current registry's Timeout value > now
		// then this server will be treated as an alive server
		if registry.Timeout == 0 || serverInfo.lastUpdateTime.Add(registry.Timeout).After(time.Now()) {
			aliveServerList = append(aliveServerList, serverAddress)
		} else {
			delete(registry.RpcServerAddressToServerInfoMap, serverAddress)
		}
	}
	sort.Strings(aliveServerList)
	log.Printf("RPC registry -> getAliveServerList: RPC registry %s return list of aliveServerList server instance (and deleted any expire server instances)...", registry.RegistryAddress)
	return aliveServerList
}

// ServeHTTP is the registry HTTP endpoint that runs at /_srpc_/registry
func (registry *Registry) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.Method {
	//GET returns all the alive servers instance currently available in the registry, used "GET-SpartanRPC-AliveServers" as customized HTTP header field
	case "GET":
		log.Printf("RPC registry -> ServeHTTP: RPC registry %s serving HTTP GET request from RPC client...", registry.RegistryAddress)
		// keep it simple, server is in request.Header
		responseWriter.Header().Set("GET-SpartanRPC-AliveServers", strings.Join(registry.getAliveServerList(), ","))

	//POST register/send heartbeat for the server instance to the registry, used "POST-SpartanRPC-AliveServer" as customized HTTP header field
	case "POST":
		// keep it simple, server is in request.Header
		serverAddress := request.Header.Get("POST-SpartanRPC-AliveServer")
		log.Printf("RPC registry -> ServeHTTP: RPC registry %s serving HTTP POST request from RPC server %s...", registry.RegistryAddress, serverAddress)
		if serverAddress == "" {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		registry.registerServer(serverAddress)
	default:
		responseWriter.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// LaunchAndServe is a convenient approach for registry to register an individual HTTP handler and have it serve the specified path
func (registry *Registry) LaunchAndServe() {
	//log.Printf("RPC registry -> LaunchAndServe: RPC registry %s initializing an HTTP multiplexer (handler) for HTTP message receiving/sending...", registry.registryAddress)
	serverMultiplexer := http.NewServeMux()
	serverMultiplexer.HandleFunc(DefaultRegistryPath, registry.ServeHTTP)
	log.Printf("RPC registry -> LaunchAndServe: RPC registry %s finished initializing the HTTP multiplexer (handler), and it is serving on URL path: %s", registry.RegistryAddress, registry.RegistryURL)
	_ = http.Serve(registry.Listener, serverMultiplexer)
}

/*
//var DefaultRegister = CreateRegistry(DefaultURL, DefaultTimeout)

func LaunchAndServe() {
	DefaultRegister.LaunchAndServe(defaultPath)
}
*/
