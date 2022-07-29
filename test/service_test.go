package test

import (
	"Distributed-RPC-Framework/service"
	"reflect"
	"testing"
)

func TestCreateService(t *testing.T) {
	//var arithmetic Arithmetic
	t.Helper()
	var demo Demo
	//newService := service.CreateService(&arithmetic)
	newService := service.CreateService(&demo)
	_assert(len(newService.ServiceMethod) == 1, "Wrong service method length, expect 1, but got %d", len(newService.ServiceMethod))
	methodType := newService.ServiceMethod["Addition_demo"]
	_assert(methodType != nil, "Wrong service method type, Sum expected to be type Sum, but got type nil")
}

func TestMethodType_Call(t *testing.T) {
	//var arithmetic Arithmetic
	t.Helper()
	var demo Demo
	newService := service.CreateService(&demo)
	methodType := newService.ServiceMethod["Addition_demo"]

	input := methodType.CreateInput()
	output := methodType.CreateOutput()
	input.Set(reflect.ValueOf(Input{A: 3, B: 4}))
	err := newService.Call(methodType, input, output)
	_assert(err == nil, "Method call error: err expected to be nil, but got %s", err)
	_assert(*output.Interface().(*int) == 7, "Method call error: output expected to be 7, but got %d", *output.Interface().(*int))
	_assert(methodType.CallCounts() == 1, "Method call error: callCount expected to be 1, but got %s", methodType.CallCounts())
}
