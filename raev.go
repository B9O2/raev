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
	mTrans  func(f func(args ...Object) ([]Object, error)) (Method, error)
	cInit   func(name string) (Class, error)
	cTrans  func(f func(kwargs map[string]Object) (Class, error)) (Method, error)
	sInits  map[string]func(kwargs map[string]Object) (ret Class, err error)
}

func (r *Raev) ValueTransfer(value any) (obj Object, err error) {
	obj = r.zeroObj
	defer func() {
		if rec := recover(); rec != nil {
			err = errors.New("Raev::ValueTransfer::Panic@" + fmt.Sprint(rec) + reflect.TypeOf(value).String())
		}
	}()

	if value == nil {
		return
	}

	v := reflect.ValueOf(value)

	//already registered
	if initF, ok := r.sInits[v.Type().String()]; ok {
		fields := map[string]Object{}
		for i := 0; i < v.Elem().Type().NumField(); i++ {
			tf := v.Elem().Type().Field(i)
			vf := v.Elem().Field(i)
			object, err := r.ValueTransfer(vf.Interface())
			if err != nil {
				return nil, err
			}
			fields[tf.Name] = object
		}
		obj, err = initF(fields)
	} else {
		obj, err = r.vTrans(value)
	}
	return obj, err
}

func (r *Raev) ObjectTransfer(obj Object, expected reflect.Type) (reflect.Value, error) {
	transfer, err := r.oTrans(obj)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	if transfer != nil {
		return reflect.ValueOf(transfer), nil
	} else {
		return reflect.New(expected).Elem(), nil
	}
}

// RegisterClassTransfer 注册类转换方法
func (r *Raev) RegisterClassTransfer(classInit func(name string) (Class, error), classTransfer func(f func(kwargs map[string]Object) (Class, error)) (Method, error)) {
	r.cInit = classInit
	r.cTrans = classTransfer
}

func (r *Raev) NewMethod(vm reflect.Value) (Method, error) {
	f := func(margs ...Object) (mret []Object, err error) {
		var arguments []reflect.Value
		mret = []Object{r.zeroObj}
		defer func() {
			if rec := recover(); rec != nil {
				err = errors.New("Raev::Method::Panic@" + fmt.Sprint(rec))
			}
		}()

		l := len(margs)
		for i := 0; i < vm.Type().NumIn(); i++ {
			inType := vm.Type().In(i)
			if i >= l {
				arguments = append(arguments, reflect.New(inType).Elem())
			} else {
				value, err := r.ObjectTransfer(margs[i], inType)
				if err != nil {
					return nil, err
				}
				arguments = append(arguments, value)
			}
		}

		rets := vm.Call(arguments)
		lr := len(rets)
		if lr > 0 {
			if errObj := rets[lr-1]; errObj.Type().Name() == "error" {
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
				obj, err := r.ValueTransfer(retV.Interface())
				if err != nil {
					return nil, err
				}
				mret = append(mret, obj)
			}
		}
		return
	}
	return r.mTrans(f)
}

func (r *Raev) NewClass(name string, source any, defaultArgs map[string]any) (_ Method, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = errors.New("Raev::" + name + "(class)::Panic@" + fmt.Sprint(rec))
		}
	}()
	if r.cInit == nil || r.cTrans == nil {
		return nil, errors.New("Raev::" + name + "(class)::Panic@the class transfer not be registered")
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
			f := gos.FieldByName(tf.Name)
			if !f.CanInterface() {
				continue
			}
			if da, ok := defaultArgs[tf.Name]; ok {
				if obj, err := r.ValueTransfer(da); err == nil {
					f.Set(reflect.ValueOf(da))
					c.SetMemberVar(tf.Name, obj)
				} else {
					return nil, err
				}
			} else {
				c.SetMemberVar(tf.Name, r.zeroObj)
			}
		}

		//Set init
		for key, obj := range kwargs {
			f := gos.FieldByName(key)
			//exist and exported
			if f.IsValid() && f.CanInterface() {
				if v, err := r.ObjectTransfer(obj, f.Type()); err != nil {
					return nil, err
				} else {
					f.Set(v)
					c.SetMemberVar(key, obj)
				}
			} else {
				return nil, errors.New("<init> unnecessary argument '" + key + "'")
			}
		}

		for i := 0; i < pt.NumMethod(); i++ {
			mt := pt.Method(i)
			mv := pGos.Method(i)
			if function, err := r.NewMethod(mv); err == nil {
				c.SetMethod(mt.Name, function)
			} else {
				return nil, err
			}
		}
		ret = c
		return
	}
	r.sInits[reflect.TypeOf(source).String()] = initF
	return r.cTrans(initF)
}

func NewRaev(zero Object, objectTransfer func(Object) (any, error), valueTransfer func(any) (Object, error), methodTransfer func(func(...Object) ([]Object, error)) (Method, error)) *Raev {
	rv := &Raev{
		sInits: map[string]func(kwargs map[string]Object) (ret Class, err error){},
	}
	rv.zeroObj = zero
	rv.vTrans = valueTransfer
	rv.oTrans = objectTransfer
	rv.mTrans = methodTransfer
	return rv
}
