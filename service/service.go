package service

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// TODO: const

// TODO: struct
type ServiceType struct {
	service    reflect.Method
	inputType  reflect.Type
	returnType reflect.Type
	countCalls uint64
}

type Service struct {
	ServiceName   string
	serviceType   reflect.Type
	serviceValue  reflect.Value
	ServiceMethod map[string]*ServiceType
}

//TODO: function

func (service *ServiceType) CountCalls() uint64 {
	return atomic.LoadUint64(&service.countCalls)
}

func (service *ServiceType) GetInput() reflect.Value {
	var input reflect.Value
	switch service.inputType.Kind() {
	case reflect.Ptr:
		input = reflect.New(service.inputType.Elem())
	default:
		input = reflect.New(service.inputType).Elem()
	}
	return input
}

func (service *ServiceType) GetOutput() reflect.Value {
	// reply must be a pointer type
	output := reflect.New(service.returnType.Elem())
	if service.returnType.Elem().Kind() == reflect.Map {
		output.Elem().Set(reflect.MakeMap(service.returnType.Elem()))
	} else if service.returnType.Elem().Kind() == reflect.Slice {
		output.Elem().Set(reflect.MakeSlice(service.returnType.Elem(), 0, 0))
	} else {
		log.Println("Server - getOutput error: error reflect return type.")
	}
	return output
}

func CreateService(ServiceValue interface{}) *Service {
	Service := new(Service)
	Service.serviceValue = reflect.ValueOf(ServiceValue)
	Service.ServiceName = reflect.Indirect(Service.serviceValue).Type().Name()
	if !ast.IsExported(Service.ServiceName) {
		log.Fatalf("service - create service error: service name: %s is invalid ", Service.ServiceName)
	}
	Service.serviceType = reflect.TypeOf(ServiceValue)
	Service.createMethod()
	return Service
}

func (Service *Service) createMethod() {
	Service.ServiceMethod = make(map[string]*ServiceType)
	for i := 0; i < Service.serviceType.NumMethod(); i++ {
		method := Service.serviceType.Method(i)
		methodType := method.Type
		switch{
		case methodType.NumIn() != 1:
			log.Println("Service - createMethod error: methodType.NumIn() != 1")
			continue
		case methodType.NumOut() != 3:
			log.Println("Service - createMethod error: methodType.NumIn() != 3")
			continue
		case methodType.Out(0) != reflect.TypeOf((*error)(nil)).Elem():
			log.Println("Service - createMethod error: methodType.Out(0) != reflect.TypeOf((*error)(nil)).Elem()")
			continue
		default:
			inputType, outputType := method.Type.In(1), method.Type.In(2)
			if !(ast.IsExported(inputType.Name()) || inputType.PkgPath() == "") {
				log.Println("Service - createMethod error: !(ast.IsExported(inputType.Name())|| inputType.PkgPath() == '')")
				continue
			} else if !(ast.IsExported(outputType.Name()) || outputType.PkgPath() == "") {
				log.Println("Service - createMethod error: !(ast.IsExported(outputType.Name())|| outputType.PkgPath() == '')")
				continue
			}
			Service.ServiceMethod[method.Name] = &ServiceType{
				service:    method,
				inputType:  inputType,
				returnType: outputType,
			}
			log.Printf("RPC server: created %s.%s\n", Service.ServiceName, method.Name)
		}
	}
}

func (Service *Service) Call(ServiceType *ServiceType, input, output reflect.Value) error {
	atomic.AddUint64(&ServiceType.countCalls, 1)
	Function := ServiceType.service.Func
	outputValue := Function.Call([]reflect.Value{Service.serviceValue, input, output})
	errors := outputValue[0].Interface()
	if errors != nil {
		return errors.(error)
	} else {
		log.Println("finished service Call")
		return nil
	}
}
