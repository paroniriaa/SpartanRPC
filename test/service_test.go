package test

import (
	"Distributed-RPC-Framework/service"
	"log"
	"reflect"
	"testing"
)

type Test int

type Input struct{ Number1, Number2 int }

func (function Test) Sum(input Input, output *int) error {
	*output = input.Number1 + input.Number2
	return nil
}

/*
func (function Test) sum(args Input, reply *int) error {
	*reply = args.Number1 + args.Number2
	return nil
}
*/

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		log.Fatalf("assertion failed: "+msg, v...)
	}
}

func TestCreateService(t *testing.T) {
	var test Test
	newService := service.CreateService(&test)
	_assert(len(newService.ServiceMethod) == 1, "Wrong service method length, expect 1, but got %d", len(newService.ServiceMethod))
	methodType := newService.ServiceMethod["Sum"]
	_assert(methodType != nil, "Wrong service method type, Sum expected to be type Sum, but got type nil")
}

func TestMethodType_Call(t *testing.T) {
	var test Test
	newService := service.CreateService(&test)
	methodType := newService.ServiceMethod["Sum"]

	input := methodType.CreateInput()
	output := methodType.CreateOutput()
	input.Set(reflect.ValueOf(Input{Number1: 3, Number2: 4}))
	err := newService.Call(methodType, input, output)
	_assert(err == nil, "Method call error: err expected to be nil, but got %s", err)
	_assert(*output.Interface().(*int) == 7, "Method call error: output expected to be 7, but got %d", *output.Interface().(*int))
	_assert(methodType.CallCounts() == 1, "Method call error: callCount expected to be 1, but got %s", methodType.CallCounts())
}
