package test

import (
	"Distributed-RPC-Framework/service"
	"log"
	"reflect"
	"testing"
)

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		log.Fatalf("assertion failed: "+msg, v...)
	}
}

func TestCreateService(t *testing.T) {
	var arithmetic Arithmetic
	newService := service.CreateService(&arithmetic)
	_assert(len(newService.ServiceMethod) == 1, "Wrong service method length, expect 1, but got %d", len(newService.ServiceMethod))
	methodType := newService.ServiceMethod["Sum"]
	_assert(methodType != nil, "Wrong service method type, Sum expected to be type Sum, but got type nil")
}

func TestMethodType_Call(t *testing.T) {
	var arithmetic Arithmetic
	newService := service.CreateService(&arithmetic)
	methodType := newService.ServiceMethod["Addition"]

	input := methodType.CreateInput()
	output := methodType.CreateOutput()
	input.Set(reflect.ValueOf(Input{A: 3, B: 4}))
	err := newService.Call(methodType, input, output)
	_assert(err == nil, "Method call error: err expected to be nil, but got %s", err)
	_assert(*output.Interface().(*int) == 7, "Method call error: output expected to be 7, but got %d", *output.Interface().(*int))
	_assert(methodType.CallCounts() == 1, "Method call error: callCount expected to be 1, but got %s", methodType.CallCounts())
}