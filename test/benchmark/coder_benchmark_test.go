package benchmark

import (
	"Distributed-RPC-Framework/coder"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func BenchmarkCoder(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	time.Sleep(time.Second)

	b.Run("CoderMapInitialization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			TestNewCoderFuncMap := make(map[coder.CoderType]coder.CoderInitializer)
			TestNewCoderFuncMap[coder.Json] = coder.NewJsonCoder
		}
	})
}
