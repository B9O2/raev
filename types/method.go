package types

import (
	"errors"
	"reflect"
)

type Method struct {
	name         string
	params, rets []Parameter
	f            func([]ExtendObject) ([]ExtendObject, error)
}

func (m *Method) Params() []Parameter {
	return m.params
}

func (m *Method) Name() string {
	return m.name
}

func (m *Method) SetParameters(params []Parameter) {
	m.params = params
}

func (m *Method) SetReturnParams(params []Parameter) {
	m.rets = params
}

func (m *Method) ReturnParams() []Parameter {
	return m.rets
}

func (m *Method) Raw() func([]ExtendObject) ([]ExtendObject, error) {
	return m.f
}

func (m *Method) Call(args []ExtendObject) (res []ExtendObject, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(err.Error())
		}
	}()
	return m.f(args)
}

func NewMethod(name string, vm reflect.Value, f func([]ExtendObject) ([]ExtendObject, error)) Method {
	m := Method{
		name: name,
		f:    f,
	}
	for i := 0; i < vm.Type().NumIn(); i++ {
		m.params = append(m.params, NewParameter(vm.Type().In(i)))
	}
	for i := 0; i < vm.Type().NumOut(); i++ {
		m.rets = append(m.rets, NewParameter(vm.Type().Out(i)))
	}
	return m
}
