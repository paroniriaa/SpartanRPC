package benchmark

import (
	"Distributed-RPC-Framework/registry"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func BenchmarkRegistry(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	time.Sleep(time.Second)

	//create test needed variables
	var waitGroup sync.WaitGroup
	var testRegistry *registry.Registry
	var registryListener net.Listener
	var err error

	//create registry for testing
	registryChannel := make(chan *registry.Registry)
	waitGroup.Add(1)
	go createRegistry(":8031", registryChannel, &waitGroup)
	testRegistry = <-registryChannel
	waitGroup.Wait()

	//create a fake server address for httpRequestPOST.Header.Set() server address parameter
	serverListener, err := net.Listen("tcp", ":9031")
	if err != nil {
		log.Fatal("BenchmarkRegistry -> main error: RPC server network issue:", err)
	}
	serverAddress := serverListener.Addr().Network() + "@" + serverListener.Addr().String()

	//simulate "POST" request to ServeHTTP(), which trigger ServeHTTP() "POST" case request handling
	httpClient := &http.Client{}
	httpRequestPOST, _ := http.NewRequest("POST", testRegistry.RegistryURL, nil)
	httpRequestPOST.Header.Set("POST-SpartanRPC-AliveServer", serverAddress)

	b.Run("CreateRegistry", func(b *testing.B) {
		registryListener, err = net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal("BenchmarkRegistry -> main error: RPC registry network issue:", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registry.CreateRegistry(registryListener, registry.DefaultTimeout)
		}
	})

	//RPC registry's LaunchAndServe() is blocking, no need to test
	//b.Run("LaunchAndServe", func(b *testing.B) {})

	//httpClient.Do(httpRequestPOST) -> ServeHTTP() -> "POST" case handling
	b.Run("ServeHTTP.POST", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = httpClient.Do(httpRequestPOST)
		}
	})

	//http.Get() -> ServeHTTP() -> "GET" case handling
	b.Run("ServeHTTP.GET", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = http.Get(testRegistry.RegistryURL)
		}
	})

}
