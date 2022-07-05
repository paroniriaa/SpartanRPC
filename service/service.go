package service

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// TODO: const

// TODO: struct
type Method struct {
	methodType reflect.Method
	InputType  reflect.Type
	OutputType reflect.Type
	callCounts uint64
}

type Service struct {
	ServiceName   string
	serviceType   reflect.Type
	serviceValue  reflect.Value
	ServiceMethod map[string]*Method
}

//TODO: function

func (method *Method) CallCounts() uint64 {
	return atomic.LoadUint64(&method.callCounts)
}

func (method *Method) CreateInput() reflect.Value {
	// input can be either pointer type or value type
	var input reflect.Value
	switch method.InputType.Kind() {
	case reflect.Ptr:
		input = reflect.New(method.InputType.Elem())
	default:
		input = reflect.New(method.InputType).Elem()
	}
	return input
}

func (method *Method) CreateOutput() reflect.Value {
	// output must be a pointer type
	output := reflect.New(method.OutputType.Elem())
	if method.OutputType.Elem().Kind() == reflect.Map {
		output.Elem().Set(reflect.MakeMap(method.OutputType.Elem()))
	} else if method.OutputType.Elem().Kind() == reflect.Slice {
		output.Elem().Set(reflect.MakeSlice(method.OutputType.Elem(), 0, 0))
	} else {
		log.Printf("CreateOutput error: reflect return type should be either reflect.Map or reflect.Slice, but got %s.", method.OutputType.Elem().Kind())
	}
	return output
}

func CreateService(serviceValue interface{}) *Service {
	newService := new(Service)
	newService.serviceValue = reflect.ValueOf(serviceValue)
	newService.ServiceName = reflect.Indirect(newService.serviceValue).Type().Name()
	if !ast.IsExported(newService.ServiceName) {
		log.Fatalf("methodType - create methodType error: methodType name: %s is invalid ", newService.ServiceName)
	}
	newService.serviceType = reflect.TypeOf(serviceValue)
	newService.createMethod()
	return newService
}

func (service *Service) createMethod() {
	service.ServiceMethod = make(map[string]*Method)
	for i := 0; i < service.serviceType.NumMethod(); i++ {
		method := service.serviceType.Method(i)
		methodType := method.Type
		switch {
		case methodType.NumIn() != 1:
			log.Println("service - createMethod error: methodType.NumIn() != 1")
			continue
		case methodType.NumOut() != 3:
			log.Println("service - createMethod error: methodType.NumIn() != 3")
			continue
		case methodType.Out(0) != reflect.TypeOf((*error)(nil)).Elem():
			log.Println("service - createMethod error: methodType.Out(0) != reflect.TypeOf((*error)(nil)).Elem()")
			continue
		default:
			inputType, outputType := methodType.In(1), methodType.In(2)
			if !(ast.IsExported(inputType.Name()) || inputType.PkgPath() == "") {
				log.Println("service - createMethod error: !(ast.IsExported(inputType.Name())|| inputType.PkgPath() == '')")
				continue
			} else if !(ast.IsExported(outputType.Name()) || outputType.PkgPath() == "") {
				log.Println("service - createMethod error: !(ast.IsExported(outputType.Name())|| outputType.PkgPath() == '')")
				continue
			}
			service.ServiceMethod[method.Name] = &Method{
				methodType: method,
				InputType:  inputType,
				OutputType: outputType,
			}
			log.Printf("RPC server: created %s.%s\n", service.ServiceName, method.Name)
		}
	}
}

func (service *Service) Call(method *Method, input, output reflect.Value) error {
	atomic.AddUint64(&method.callCounts, 1)
	function := method.methodType.Func
	outputValue := function.Call([]reflect.Value{service.serviceValue, input, output})
	errors := outputValue[0].Interface()
	if errors != nil {
		return errors.(error)
	} else {
		log.Println("finished methodType Call")
		return nil
	}
}