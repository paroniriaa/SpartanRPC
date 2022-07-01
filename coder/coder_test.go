package coder

import (
	"reflect"
	"testing"
)

func TestInit(test *testing.T) {
	test.Helper()
	TestNewCoderFuncMap := make(map[Type]NewCoderFunc)
	TestNewCoderFuncMap[Json] = NewJsonCoder
	Init()
	if reflect.TypeOf(NewCoderFuncMap) != reflect.TypeOf(TestNewCoderFuncMap) {
		test.Errorf("NewCoderFuncMap expected be in type of %s, but got %s", reflect.TypeOf(TestNewCoderFuncMap), reflect.TypeOf(NewCoderFuncMap))
	}
	if reflect.TypeOf(NewCoderFuncMap[Json]) != reflect.TypeOf(TestNewCoderFuncMap[Json]) {
		test.Errorf("NewCoderFuncMap[Json] expected be in type of %s, but got %s", reflect.TypeOf(NewCoderFuncMap[Json]), reflect.TypeOf(TestNewCoderFuncMap[Json]))
	}
}
