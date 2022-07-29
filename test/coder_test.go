package test

import (
	"Distributed-RPC-Framework/coder"
	"reflect"
	"testing"
)

func TestInit(test *testing.T) {
	test.Helper()
	TestNewCoderFuncMap := make(map[coder.CoderType]coder.CoderInitializer)
	TestNewCoderFuncMap[coder.Json] = coder.NewJsonCoder
	if reflect.TypeOf(coder.CoderInitializerMap) != reflect.TypeOf(TestNewCoderFuncMap) {
		test.Errorf("CoderInitializerMap expected be in type of %s, but got %s", reflect.TypeOf(TestNewCoderFuncMap), reflect.TypeOf(coder.CoderInitializerMap))
	}
	if reflect.TypeOf(coder.CoderInitializerMap[coder.Json]) != reflect.TypeOf(TestNewCoderFuncMap[coder.Json]) {
		test.Errorf("CoderInitializerMap[Json] expected be in type of %s, but got %s", reflect.TypeOf(coder.CoderInitializerMap[coder.Json]), reflect.TypeOf(TestNewCoderFuncMap[coder.Json]))
	}
}
