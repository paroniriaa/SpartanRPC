package test

import "errors"

type Input struct {
	A, B int
}

type Output struct {
	C int
}

type Demos int

func (t Demos) Addition_demo(input Input, output *int) error {
	*output = input.A + input.B
	return nil
}


type Arithmetic int

func (t *Arithmetic) Addition(input *Input, output *Output) error {
	output.C = input.A + input.B
	return nil
}

func (t *Arithmetic) Subtraction(input *Input, output *Output) error {
	output.C = input.A - input.B
	return nil
}

func (t *Arithmetic) Multiplication(input *Input, output *Output) error {
	output.C = input.A * input.B
	return nil
}

func (t *Arithmetic) Division(input *Input, output *Output) error {
	if input.B == 0 {
		return errors.New("divide by zero")
	}
	output.C = input.A / input.B
	return nil
}

func (t *Arithmetic) Error(input *Input, output *Output) error {
	panic("ERROR")
}

type BuiltinType struct{}

func (t *BuiltinType) Pointer(input int, output *int) error {
	*output = input
	return nil
}

func (t *BuiltinType) Slice(input int, output *[]int) error {
	*output = append(*output, input)
	return nil
}

func (t *BuiltinType) Array(input int, output *[1]int) error {
	(*output)[0] = input
	return nil
}

func (t *BuiltinType) Map(input int, output *map[int]int) error {
	(*output)[input] = input
	return nil
}
