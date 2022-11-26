package unit

import (
	"Distributed-RPC-Framework/coder"
	"log"
	"reflect"
	"testing"
)

func TestCoder(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	t.Run("CoderInitialization", func(t *testing.T) {
		TestNewCoderFuncMap := make(map[coder.CoderType]coder.CoderInitializer)
		TestNewCoderFuncMap[coder.Json] = coder.NewJsonCoder
		if reflect.TypeOf(coder.CoderInitializerMap) != reflect.TypeOf(TestNewCoderFuncMap) {
			t.Errorf("CoderInitialization error: CoderInitializerMap expected be in type of %s, but got %s", reflect.TypeOf(TestNewCoderFuncMap), reflect.TypeOf(coder.CoderInitializerMap))
		}
		if reflect.TypeOf(coder.CoderInitializerMap[coder.Json]) != reflect.TypeOf(TestNewCoderFuncMap[coder.Json]) {
			t.Errorf("CoderInitialization error: CoderInitializerMap[Json] expected be in type of %s, but got %s", reflect.TypeOf(coder.CoderInitializerMap[coder.Json]), reflect.TypeOf(TestNewCoderFuncMap[coder.Json]))
		}
	})
}
