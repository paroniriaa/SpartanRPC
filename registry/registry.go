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
	DefaultPath    = "/_srpc_/registry"
	DefaultURL     = "http://localhost:9999/_srpc_/registry"
	DefaultPort    = ":9999"
	DefaultTimeout = time.Minute * 5
)

// CreateRegistry create a registry instance with Timeout setting
func CreateRegistry(listener net.Listener, timeout time.Duration) (*Registry, error) {
	log.Printf("RPC registry -> CreateRegistry: creating RPC registry on port %s...", listener.Addr().String())
	if listener == nil {
		return nil, errors.New("RPC server > CreateServer error: Network listener should not be nil, but received nil")
	}
	//the port parameter passed-in will be in the form of "[::]:1234", so we need to extract port
	registryURL := "http://localhost" + listener.Addr().String()[4:] + DefaultPath
	log.Printf("RPC registry -> CreateRegistry: created RPC registry on HTTP end-point %s...", registryURL)
	return &Registry{
		Listener:                        listener,
		RegistryURL:                     registryURL,
		RpcServerAddressToServerInfoMap: make(map[string]*ServerInfo),
		Timeout:                         timeout,
	}, nil
}

func (registry *Registry) registerServer(addr string) {
	log.Println("RPC registry -> registerServer: RPC registry registering server instance...")
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	serverInfo := registry.RpcServerAddressToServerInfoMap[addr]
	if serverInfo == nil {
		registry.RpcServerAddressToServerInfoMap[addr] = &ServerInfo{ServerAddress: addr, lastUpdateTime: time.Now()}
	} else {
		// if the server already exists, update its lastUpdateTime time as now to keep it alive
		serverInfo.lastUpdateTime = time.Now()
	}
}

func (registry *Registry) getAliveServerList() []string {
	log.Println("RPC registry -> getAliveServerList: RPC registry return list of aliveServerList server instance (and delete any expire server instances)...")
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
	return aliveServerList
}

// ServeHTTP is the registry HTTP endpoint that runs at /_srpc_/registry
func (registry *Registry) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.Method {
	//GET returns all the alive servers instance currently available in the registry, used "GET-SpartanRPC-AliveServers" as customized HTTP header field
	case "GET":
		log.Println("RPC registry -> ServeHTTP: RPC registry serving HTTP GET request...")
		// keep it simple, server is in request.Header
		responseWriter.Header().Set("GET-SpartanRPC-AliveServers", strings.Join(registry.getAliveServerList(), ","))

	//POST register/send heartbeat for the server instance to the registry, used "POST-SpartanRPC-AliveServer" as customized HTTP header field
	case "POST":
		log.Println("RPC registry -> ServeHTTP: RPC registry serving HTTP POST request...")
		// keep it simple, server is in request.Header
		serverAddress := request.Header.Get("POST-SpartanRPC-AliveServer")
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
	log.Println("RPC registry -> LaunchAndServe: RPC registry initializing an HTTP multiplexer (handler) for message receiving/sending...")
	serverMultiplexer := http.NewServeMux()
	serverMultiplexer.HandleFunc(DefaultPath, registry.ServeHTTP)
	log.Println("RPC registry -> LaunchAndServe: RPC registry finished initializing the HTTP multiplexer (handler), and it is serving on URL path: ", registry.RegistryURL, "")
	_ = http.Serve(registry.Listener, serverMultiplexer)
}

/*
//var DefaultRegister = CreateRegistry(DefaultURL, DefaultTimeout)

func LaunchAndServe() {
	DefaultRegister.LaunchAndServe(defaultPath)
}
*/
