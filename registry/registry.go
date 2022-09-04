package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	timeout time.Duration
	lock      sync.Mutex // protect following
	serverList map[string]*ServerInformation
}

type ServerInformation struct {
	Address  string
	InitTime time.Time
}

const (
	defaultPath    = "./registry"
	defaultTimeout = time.Minute * 5
)

// New create a registry instance with timeout setting
func NewRegistry(timeout time.Duration) *Registry {
	return &Registry{
		serverList: make(map[string]*ServerInformation),
		timeout: timeout,
	}
}

var DefaultRegister = NewRegistry(defaultTimeout)

func (reg *Registry) addServer(address string) {
	reg.lock.Lock()
	server := reg.serverList[address]
	if server == nil {
		reg.serverList[address] = &ServerInformation{Address: address, InitTime: time.Now()}
	} else {
		server.InitTime = time.Now()
	}
	defer reg.lock.Unlock()
}

func (reg *Registry) aliveServerList() []string {
	reg.lock.Lock()
	var result []string
	for address, server := range reg.serverList {
		if server.InitTime.Add(reg.timeout).Before(time.Now()) {
			if reg.timeout != 0 {
				delete(reg.serverList, address)
			}
		} else {
			result = append(result, address)
		}
	}
	sort.Strings(result)
	defer reg.lock.Unlock()
	return result
}

func (reg *Registry) ServeHTTP(context http.ResponseWriter, request *http.Request) {

	if request.Method == "POST" {
		address := request.Header.Get("RPC-Server")
		if address == "" {
			context.WriteHeader(http.StatusInternalServerError)
			return
		}
		reg.addServer(address)
	} else if request.Method == "GET" {
		context.Header().Set("RPC-Servers", strings.Join(reg.aliveServerList(), ","))
	} else {
		context.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for GeeRegistry messages on registryPath
func (reg *Registry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, reg)
	log.Println("RPC registry path:", registryPath)
}

func HandleHTTP() {
	DefaultRegister.HandleHTTP(defaultPath)
}

func heartbeatMonitoring(registry, address string) error {
	log.Println(address, "sending heart beat to registry", registry)
	httpClient := &http.Client{}
	request, _ := http.NewRequest("POST", registry, nil)
	request.Header.Set("RPC-Server", address)
	if _, err := httpClient.Do(request); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}


func Heartbeat(registry, address string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = heartbeatMonitoring(registry, address)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = heartbeatMonitoring(registry, address)
		}
	}()
}
