package test

import (
	"Distributed-RPC-Framework/loadBalancer"
	"Distributed-RPC-Framework/registry"
	"Distributed-RPC-Framework/server"
	"log"
	"net/http"
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestRegistry(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	var waitGroup sync.WaitGroup
	//create service type (arithmetic is enough)
	var arithmetic Arithmetic
	var builtinType BuiltinType
	servicesA := []any{
		&arithmetic,
	}
	servicesB := []any{
		&builtinType,
	}

	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	//LaunchAndServe() called implicitly in the go function createRegistry()
	go createRegistry(":0", registryChannel, &waitGroup)
	testRegistry := <-registryChannel
	testRegistryAddress := testRegistry.RegistryURL
	log.Printf("Registry test -> main: Registry address fetched: %s", testRegistry.RegistryURL)
	waitGroup.Wait()

	//Create registry test
	t.Run("CreateRegistry", func(t *testing.T) {
		//testRegistry pre-created
		if reflect.TypeOf(testRegistry) != reflect.TypeOf(&registry.Registry{}) {
			t.Errorf("registryA expected be in type of %s, but got %s", reflect.TypeOf(registry.Registry{}), reflect.TypeOf(testRegistry))
		}
		if reflect.TypeOf(testRegistry.RpcServerAddressToServerInfoMap) != reflect.TypeOf(make(map[string]*registry.ServerInfo)) {
			t.Errorf("registryA.RpcServerAddressToServerInfoMap expected be in type of %s, but got %s", reflect.TypeOf(registry.Registry{}), reflect.TypeOf(testRegistry))
		}
		if testRegistry.RegistryURL != testRegistryAddress {
			t.Errorf("registryA.RegistryURL expected be %s, but got %s", registry.DefaultURL, testRegistry.RegistryURL)
		}
		if testRegistry.Timeout != registry.DefaultTimeout {
			t.Errorf("registryA.Timeout expected be %s, but got %s", registry.DefaultTimeout, testRegistry.Timeout)
		}
	})

	//Handle HTTP test
	t.Run("LaunchAndServe", func(t *testing.T) {
		//testRegistry pre-created, and hosted on the default address
		_, err := http.Get(testRegistry.RegistryURL)
		if err != nil {
			t.Errorf("http.Get() expect no error, but got error: %s", err)
		}
	})

	//ServerHTTP POST case test
	t.Run("ServeHTTP.POST", func(t *testing.T) {
		//testRegistry pre-created, and its server infos should be empty
		if len(testRegistry.RpcServerAddressToServerInfoMap) != 0 {
			t.Errorf("Before Heartbeat(), len(testRegistry.RegistryURL) expected be %d, but got %d", 0, len(testRegistry.RpcServerAddressToServerInfoMap))
		}

		//create 2 servers and register to the registry
		serverChannelA := make(chan *server.Server)
		serverChannelB := make(chan *server.Server)
		waitGroup.Add(2)
		go createServerOnRegistry(":0", testRegistry.RegistryURL, servicesA, serverChannelA, &waitGroup)
		testServerA := <-serverChannelA
		log.Printf("Registry test -> main: Server A (registered) address fetched: %s", testServerA.ServerAddress)
		go createServerOnRegistry(":0", testRegistry.RegistryURL, servicesB, serverChannelB, &waitGroup)
		testServerB := <-serverChannelB
		log.Printf("Registry test -> main: Server B (registered) address fetched: %s", testServerB.ServerAddress)
		waitGroup.Wait()

		//Heartbeat() -> http.NewRequest("POST") -> send POST message -> ServeHTTP() -> POST case handling.
		testServerA.Heartbeat(testRegistry.RegistryURL, 0)
		testServerB.Heartbeat(testRegistry.RegistryURL, 0)

		if len(testRegistry.RpcServerAddressToServerInfoMap) != 2 {
			t.Errorf("After Heartbeat(), len(testRegistry.RegistryURL) expected be %d, but got %d", 2, len(testRegistry.RpcServerAddressToServerInfoMap))
		}
		if testRegistry.RpcServerAddressToServerInfoMap[testServerA.ServerAddress].ServerAddress != testServerA.ServerAddress {
			t.Errorf("testRegistry.RpcServerAddressToServerInfoMap[testServerA.ServerAddress].ServerAddress expected be %s, but got %s", testServerA.ServerAddress, testRegistry.RpcServerAddressToServerInfoMap[testServerA.ServerAddress].ServerAddress)
		}
		if testRegistry.RpcServerAddressToServerInfoMap[testServerB.ServerAddress].ServerAddress != testServerB.ServerAddress {
			t.Errorf("testRegistry.RpcServerAddressToServerInfoMap[testServerB.ServerAddress].ServerAddress expected be %s, but got %s", testServerB.ServerAddress, testRegistry.RpcServerAddressToServerInfoMap[testServerB.ServerAddress].ServerAddress)
		}
		//clean up current stored server infos for next test case
		delete(testRegistry.RpcServerAddressToServerInfoMap, testServerA.ServerAddress)
		delete(testRegistry.RpcServerAddressToServerInfoMap, testServerB.ServerAddress)

	})

	//ServerHTTP GET case test
	t.Run("ServeHTTP.GET", func(t *testing.T) {
		//before Heartbeat() -> create registry-side load balancer for testing
		//GetServerList() -> RefreshServerList() -> http.Get() -> send GET message -> ServeHTTP() -> GET case handling.
		registrySideLoadBalancer := loadBalancer.CreateLoadBalancerRegistrySide(testRegistry.RegistryURL, 0)
		serverList, err := registrySideLoadBalancer.GetServerList()
		//log.Printf("testRegistry: %+v", testRegistry)
		//log.Printf("serverList: %+v", serverList)

		if err != nil {
			t.Errorf("Before Heartbeat(), registrySideLoadBalancer.GetServerList() expect no error, but got error: %s", err)
		}
		if len(serverList) != 0 {
			t.Errorf("Before Heartbeat(), len(serverList) expected be %d, but got %d", 0, len(serverList))
		}

		//create 2 servers and register to the registry, Heartbeat() called implicitly, registry will be updated
		serverChannelA := make(chan *server.Server)
		serverChannelB := make(chan *server.Server)
		waitGroup.Add(2)
		go createServerOnRegistry(":0", testRegistry.RegistryURL, servicesA, serverChannelA, &waitGroup)
		testServerA := <-serverChannelA
		log.Printf("Registry test -> main: Server A (registered) address fetched: %s", testServerA.ServerAddress)
		go createServerOnRegistry(":0", testRegistry.RegistryURL, servicesB, serverChannelB, &waitGroup)
		testServerB := <-serverChannelB
		log.Printf("Registry test -> main: Server B (registered) address fetched: %s", testServerB.ServerAddress)
		waitGroup.Wait()

		//after Heartbeat() -> create registry-side load balancer for testing
		//GetServerList() -> RefreshServerList() -> http.Get() -> send GET message -> ServeHTTP() -> GET case handling.
		registrySideLoadBalancer = loadBalancer.CreateLoadBalancerRegistrySide(testRegistry.RegistryURL, 0)
		serverList, err = registrySideLoadBalancer.GetServerList()
		//log.Printf("testRegistry: %+v", testRegistry)
		//log.Printf("serverList: %+v", serverList)

		if err != nil {
			t.Errorf("After Heartbeat(), registrySideLoadBalancer.GetServerList() expect no error, but got error: %s", err)
		}
		if len(serverList) != 2 {
			t.Errorf("After Heartbeat(), len(serverList) expected be %d, but got %d", 2, len(serverList))
		}
		sort.Strings(serverList)
		if serverList[0] != testServerA.ServerAddress {
			t.Errorf("After Heartbeat(), serverList[0] expected be %s, but got %s", testServerA.ServerAddress, serverList[0])
		}
		if serverList[1] != testServerB.ServerAddress {
			t.Errorf("After Heartbeat(), serverList[0] expected be %s, but got %s", testServerB.ServerAddress, serverList[0])
		}
	})
}
