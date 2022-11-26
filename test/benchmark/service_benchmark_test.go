package benchmark

import (
	"Distributed-RPC-Framework/service"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
)

func BenchmarkService(b *testing.B) {
	//benchmark testing setup
	b.Helper()
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	//create test needed variables
	var arithmetic Arithmetic
	var testService *service.Service
	var testMethod *service.Method
	var testInput reflect.Value
	var testOutput reflect.Value

	//create service and method for testing
	testService = service.CreateService(&arithmetic)
	testMethod = testService.ServiceMethod["Addition"]

	//create input and output for testing
	testInput = testMethod.CreateInput()
	testOutput = testMethod.CreateOutput()

	//initialize input with testing value
	testInput.Elem().FieldByName("A").SetInt(0)
	testInput.Elem().FieldByName("B").SetInt(0)

	//called CreateMethod implicitly
	b.Run("CreateService", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.CreateService(&arithmetic)
		}
	})

	b.Run("CreateMethod", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testService.ServiceMethod["Addition"]
		}
	})

	b.Run("CreateInput", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testMethod.CreateInput()
		}
	})

	b.Run("CreateOutput", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testMethod.CreateOutput()
		}
	})

	b.Run("Call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testService.Call(testMethod, testInput, testOutput)
		}
	})

	b.Run("CallCounts", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = testMethod.CallCounts()
		}
	})

}
