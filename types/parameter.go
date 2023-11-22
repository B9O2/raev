package types

import "reflect"

// Parameter 解释器形参
type Parameter struct {
	reflect.Type
}

func NewParameter(t reflect.Type) Parameter {
	return Parameter{
		t,
	}
}
