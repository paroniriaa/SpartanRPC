package test

import (
	"Distributed-RPC-Framework/service"
	"log"
	"reflect"
	"testing"
)

//type Foo int

type In struct{ Num1, Num2 int }

func (f Demos) Sum(args In, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// it's not a exported Method
func (f Demos) sum(args In, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}



func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		log.Fatalf("assertion failed: "+msg, v...)
	}
}

func TestCreateService(t *testing.T) {
	var foo Demos
	newService := service.CreateService(&foo)
	_assert(len(newService.ServiceMethod) == 1, "Wrong service method length, expect 1, but got %d", len(newService.ServiceMethod))
	methodType := newService.ServiceMethod["Sum"]
	_assert(methodType != nil, "Wrong service method type, Sum expected to be type Sum, but got type nil")
}

func TestMethodType_Call(t *testing.T) {
	var foo Demos
	newService := service.CreateService(&foo)
	methodType := newService.ServiceMethod["Sum"]

	input := methodType.CreateInput()
	output := methodType.CreateOutput()
	input.Set(reflect.ValueOf(In{Num1: 3, Num2: 4}))
	err := newService.Call(methodType, input, output)
	_assert(err == nil, "Method call error: err expected to be nil, but got %s", err)
	_assert(*output.Interface().(*int) == 7, "Method call error: output expected to be 7, but got %d", *output.Interface().(*int))
	_assert(methodType.CallCounts() == 1, "Method call error: callCount expected to be 1, but got %s", methodType.CallCounts())
}