package test

import (
	"Distributed-RPC-Framework/coder"
	"reflect"
	"testing"
)

func TestInit(test *testing.T) {
	test.Helper()
	TestNewCoderFuncMap := make(map[coder.CoderType]coder.CoderFunction)
	TestNewCoderFuncMap[coder.Json] = coder.NewJsonCoder
	if reflect.TypeOf(coder.CoderFunctionMap) != reflect.TypeOf(TestNewCoderFuncMap) {
		test.Errorf("CoderFunctionMap expected be in type of %s, but got %s", reflect.TypeOf(TestNewCoderFuncMap), reflect.TypeOf(coder.CoderFunctionMap))
	}
	if reflect.TypeOf(coder.CoderFunctionMap[coder.Json]) != reflect.TypeOf(TestNewCoderFuncMap[coder.Json]) {
		test.Errorf("CoderFunctionMap[Json] expected be in type of %s, but got %s", reflect.TypeOf(coder.CoderFunctionMap[coder.Json]), reflect.TypeOf(TestNewCoderFuncMap[coder.Json]))
	}
}
