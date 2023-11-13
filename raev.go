package raev

import (
	"errors"
	"fmt"
	"reflect"
)

type Object any
type Method any
type Class interface {
	SetMemberVar(name string, obj Object)
	SetMethod(name string, method Method)
}

type Raev struct {
	zeroObj Object
	vTrans  func(value any) (Object, error)
	oTrans  func(obj Object) (any, error)
	mTrans  func(name string, f func(args ...Object) ([]Object, error)) (Method, error)
	cInit   func(name string) (Class, error)
	cTrans  func(name string, f func(kwargs map[string]Object) (Class, error)) (Method, error)
	sInits  map[string]func(kwargs map[string]Object) (ret Class, err error)
}

func (r *Raev) ValueTransfer(v any) (Object, error) {
	return r.vTrans(v)
}

func (r *Raev) ObjectTransfer(obj Object) (reflect.Value, error) {
	transfer, err := r.oTrans(obj)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	return reflect.ValueOf(transfer), nil
}

// RegisterClassTransfer 注册类转换方法
func (r *Raev) RegisterClassTransfer(classInit func(name string) (Class, error), classTransfer func(name string, f func(kwargs map[string]Object) (Class, error)) (Method, error)) {
	r.cInit = classInit
	r.cTrans = classTransfer
}

func (r *Raev) NewMethod(name string, vm reflect.Value) (Method, error) {
	f := func(margs ...Object) (mret []Object, err error) {
		var arguments []reflect.Value
		mret = []Object{r.zeroObj}
		defer func() {
			if r := recover(); r != nil {
				err = errors.New(fmt.Sprint(r))
			}
		}()

		for _, marg := range margs {
			value, err := r.ObjectTransfer(marg)
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, value)
		}
		for i, arg := range arguments {
			if !arg.IsValid() {
				arguments[i] = reflect.New(vm.Type().In(i)).Elem()
			}
		}

		al := len(arguments)
		for i := 0; i < vm.Type().NumIn(); i++ {
			if i >= al {
				arguments = append(arguments, reflect.New(vm.Type().In(i)).Elem())
			}
		}
		rets := vm.Call(arguments)
		lr := len(rets)
		if lr > 0 {
			errObj := rets[lr-1]
			if errObj.Type().Name() == "error" {
				if !errObj.IsNil() {
					return nil, errObj.Interface().(error)
				} else { //is err,but nil
					rets = rets[:lr-1]
				}
			}
		}

		if len(rets) > 0 {
			mret = []Object{}
			for _, retV := range rets {
				var obj Object
				if initF, ok := r.sInits[retV.Type().String()]; ok { //already registered
					fields := map[string]Object{}
					for i := 0; i < retV.Elem().Type().NumField(); i++ {
						tf := retV.Elem().Type().Field(i)
						vf := retV.Elem().Field(i)
						object, err := r.ValueTransfer(vf.Interface())
						if err != nil {
							return nil, err
						}
						fields[tf.Name] = object
					}
					obj, err = initF(fields)
				} else {
					obj, err = r.ValueTransfer(retV.Interface())
				}
				if err != nil {
					return nil, err
				}
				mret = append(mret, obj)
			}
		}

		return
	}
	return r.mTrans(name, f)
}

func (r *Raev) NewClass(name string, source any, defaultArgs map[string]any) (Method, error) {
	if r.cInit == nil || r.cTrans == nil {
		return nil, errors.New("raev: the class transfer not be registered")
	}
	if defaultArgs == nil {
		defaultArgs = map[string]any{}
	}

	initF := func(kwargs map[string]Object) (ret Class, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				err = errors.New(fmt.Sprint(rec))
			}
		}()
		pGos := reflect.New(reflect.TypeOf(source).Elem())
		c, err := r.cInit(name)
		if err != nil {
			return nil, err
		}
		pt := pGos.Type()
		t := pGos.Elem().Type()
		gos := pGos.Elem()
		//Set default
		for i := 0; i < gos.NumField(); i++ {
			tf := t.Field(i)
			if da, ok := defaultArgs[tf.Name]; ok {
				obj, err := r.ValueTransfer(da)
				if err != nil {
					return nil, err
				}
				c.SetMemberVar(tf.Name, obj)
				gos.FieldByName(tf.Name).Set(reflect.ValueOf(da))
			} else {
				c.SetMemberVar(tf.Name, r.zeroObj)
			}
		}

		//Set init
		for key, obj := range kwargs {
			if gos.FieldByName(key).IsValid() {
				c.SetMemberVar(key, obj)
				v, err := r.ObjectTransfer(obj)
				if err != nil {
					return nil, err
				}
				gos.FieldByName(key).Set(v)
			} else {
				return nil, errors.New("<init> unnecessary argument '" + key + "'")
			}
		}

		for i := 0; i < pt.NumMethod(); i++ {
			mt := pt.Method(i)
			mv := pGos.Method(i)
			function, err := r.NewMethod(mt.Name, mv)
			if err != nil {
				return nil, err
			}
			c.SetMethod(mt.Name, function)
		}
		ret = c
		return
	}
	r.sInits[reflect.TypeOf(source).String()] = initF
	return r.cTrans(name, initF)
}

func NewRaev(zero Object, objectTransfer func(Object) (any, error), valueTransfer func(any) (Object, error), methodTransfer func(string, func(...Object) ([]Object, error)) (Method, error)) *Raev {
	rv := &Raev{
		sInits: map[string]func(kwargs map[string]Object) (ret Class, err error){},
	}
	rv.zeroObj = zero
	rv.vTrans = valueTransfer
	rv.oTrans = objectTransfer
	rv.mTrans = methodTransfer
	return rv
}
