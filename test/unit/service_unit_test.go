package unit

import (
	"Distributed-RPC-Framework/service"
	"log"
	"reflect"
	"testing"
)

func TestService(t *testing.T) {
	t.Helper()
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//create test needed variables
	var arithmetic Arithmetic
	var testService *service.Service
	var testMethod *service.Method
	var testInput reflect.Value
	var testOutput reflect.Value
	var testCallCount uint64
	var err error

	t.Run("CreateService", func(t *testing.T) {
		testService = service.CreateService(&arithmetic)
		_assert(testService != nil, "Wrong testService, expect %s, but got %s", "not nil", "nil")
		_assert(reflect.TypeOf(testService) == reflect.TypeOf(&service.Service{}), "Wrong reflect.TypeOf(testService), expect %+v, but got %d", reflect.TypeOf(&service.Service{}), reflect.TypeOf(testService))
		_assert(testService.ServiceName == "Arithmetic", "Wrong testService.ServiceName, expect %s, but got %s", "Arithmetic", testService.ServiceName)
		_assert(reflect.TypeOf(testService.ServiceMethod) == reflect.TypeOf(service.Service{}.ServiceMethod), "Wrong reflect.TypeOf(testService.ServiceMethod), expect %+v, but got %+v", reflect.TypeOf(service.Service{}.ServiceMethod), reflect.TypeOf(testService.ServiceMethod))
		_assert(testService.ServiceMethod != nil, "Wrong testService.ServiceMethod, expect %s, but got %s", "Arithmetic", "nil")
		_assert(len(testService.ServiceMethod) == 4, "Wrong len(testService.ServiceMethod), expect %d, but got %d", 4, len(testService.ServiceMethod))

	})

	t.Run("CreateMethod", func(t *testing.T) {
		testMethod = testService.ServiceMethod["Addition"]
		_assert(testMethod != nil, "Wrong testMethod, expect %s, but got %s", "not nil", "nil")
		_assert(reflect.TypeOf(testMethod) == reflect.TypeOf(&service.Method{}), "Wrong reflect.TypeOf(testMethod), expect %+v, but got %d", reflect.TypeOf(&service.Method{}), reflect.TypeOf(testMethod))
		_assert(testMethod.MethodName == "Addition", "Wrong testService.ServiceName, expect %s, but got %s", "Addition", testMethod.MethodName)
		_assert(testMethod.InputType.Elem() == reflect.TypeOf(Input{}), "Wrong testMethod.InputType.Elem(), expect %+v, but got %+v", reflect.TypeOf(Input{}), testMethod.InputType.Elem())
		_assert(testMethod.OutputType.Elem() == reflect.TypeOf(Output{}), "Wrong testMethod.OutputType.Elem(), expect %+v, but got %+v", reflect.TypeOf(Output{}), testMethod.OutputType.Elem())
	})

	t.Run("CreateInput", func(t *testing.T) {
		testInput = testMethod.CreateInput()
		//log.Printf("testInput: %+v", testInput.Type())
		_assert(testInput.Type() == reflect.TypeOf(&Input{}), "Wrong testInput.Type(), expect %+v, but got %+v", reflect.TypeOf(&Input{}), testInput.Type())
	})

	t.Run("CreateOutput", func(t *testing.T) {
		testOutput = testMethod.CreateOutput()
		//log.Printf("testOutput: %+v", testOutput.Type())
		_assert(testOutput.Type() == reflect.TypeOf(&Output{}), "Wrong testOutput.Type(), expect %+v, but got %+v", reflect.TypeOf(&Output{}), testOutput.Type())
	})

	t.Run("Call", func(t *testing.T) {
		testInput.Elem().FieldByName("A").SetInt(3)
		testInput.Elem().FieldByName("B").SetInt(4)
		//log.Printf("testInput: %+v", testInput)
		err = testService.Call(testMethod, testInput, testOutput)
		//log.Printf("testOutput: %+v", testOutput.Elem().FieldByName("C").Int())
		_assert(err == nil, "Method call error: err expected to be nil, but got %s", err)
		_assert(testOutput.Elem().FieldByName("C").Int() == 7, "Method call error: testOutput expected to be %d, but got %d", 7, testOutput.Elem().FieldByName("C").Int())
	})

	t.Run("CallCounts", func(t *testing.T) {
		testCallCount = testMethod.CallCounts()
		_assert(testCallCount == 1, "Method call error: testCallCount expected to be %d, but got %d", 1, testMethod.CallCounts())
	})

}
