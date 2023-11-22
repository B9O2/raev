package types

import "reflect"

type Class struct {
	raw     reflect.Value
	vars    map[string]Argument
	methods map[string]Method
}

func (c *Class) SetVar(name string, a Argument) {
	c.vars[name] = a
}

func (c *Class) SetMethod(name string, m Method) {
	c.methods[name] = m
}

func (c *Class) RangeMethods(f func(Method) bool) {
	for _, method := range c.methods {
		if !f(method) {
			break
		}
	}
}

func (c *Class) RangeVars(f func(string, Argument) bool) {
	for name, arg := range c.vars {
		if !f(name, arg) {
			break
		}
	}
}

func (c *Class) Raw() reflect.Value {
	return c.raw
}

func NewClass(raw reflect.Value) Class {
	c := Class{
		raw:     raw,
		vars:    make(map[string]Argument),
		methods: make(map[string]Method),
	}
	return c
}
