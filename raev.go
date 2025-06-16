package raev

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/B9O2/raev/types"
)

type Raev struct {
	zeroObj types.ExtendObject
	trans   Transfer
}

func (r *Raev) ValueTransfer(value any) (obj types.ExtendObject, err error) {
	obj = r.zeroObj
	defer func() {
		if rec := recover(); rec != nil {
			err = errors.New("Raev::ValueTransfer::Panic@" + fmt.Sprint(rec) + reflect.TypeOf(value).String())
		}
	}()

	if value == nil {
		return
	}

	if class, err := r.newRawClass(value); err == nil {
		return r.trans.ToObject(class)
	}

	// 复合类型递归反射
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			return r.trans.MakeSlice(), nil
		}
		s := r.trans.MakeSlice()
		if s == nil {
			break
		}
		for i := 0; i < v.Len(); i++ {
			obj, err := r.ValueTransfer(v.Index(i).Interface())
			if err != nil {
				return r.zeroObj, err
			}
			s = r.trans.AppendSlice(s, obj)
		}
		value = s
	case reflect.Map:
		if v.Len() == 0 {
			return r.trans.MakeMap(), nil
		}
		m := r.trans.MakeMap()
		if m == nil {
			break
		}
		for iter := v.MapRange(); iter.Next(); {
			key, err := r.ValueTransfer(iter.Key().Interface())
			if err != nil {
				return r.zeroObj, err
			}
			mv, err := r.ValueTransfer(iter.Value().Interface())
			if err != nil {
				return r.zeroObj, err
			}
			m, err = r.trans.SetMap(m, key, mv)
			if err != nil {
				return r.zeroObj, err
			}
		}
		value = m
	}
	// case reflect.Chan:
	// 	fmt.Printf("chan %v\n", v.Interface())
	return r.trans.ToObject(value)

	//v := reflect.ValueOf(value)

	/*
		//already registered
		if initF, ok := r.sInits[v.Type().String()]; ok {
			fields := map[string]types.ExtendObject{}
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
	*/

}

func (r *Raev) ObjectTransfer(obj types.ExtendObject) (reflect.Value, error) {
	res, err := r.trans.ToValue(obj)
	if err != nil {
		return reflect.Value{}, err
	}
	if res != nil {
		return reflect.ValueOf(res), nil
	}
	return reflect.Value{}, nil
}

func (r *Raev) NewMethod(name string, m any) (types.ExtendMethod, error) {
	method, err := r.NewRawMethod(name, reflect.ValueOf(m))
	if err != nil {
		return nil, err
	}
	return r.trans.ToMethod(method)
}

func (r *Raev) NewRawMethod(name string, vm reflect.Value) (types.Method, error) {
	f := func(margs []types.ExtendObject) (mret []types.ExtendObject, err error) {
		var arguments []reflect.Value
		mret = []types.ExtendObject{r.zeroObj}
		defer func() {
			if rec := recover(); rec != nil {
				err = errors.New("Raev::Method::Panic@" + fmt.Sprint(rec))
			}
		}()

		l := len(margs)
		vmNumIn := vm.Type().NumIn()
		if l <= vmNumIn {
			for i := 0; i < vmNumIn; i++ {
				inType := vm.Type().In(i)
				if i >= l { //less arguments
					arguments = append(arguments, reflect.New(inType).Elem())
				} else {
					value, err := r.ObjectTransfer(margs[i])
					if err != nil {
						return nil, err
					}
					// if value.IsZero() {
					// 	value = reflect.New(inType).Elem()
					// }
					arguments = append(arguments, value)
				}
			}
		} else {
			if vm.Type().In(vmNumIn-1).Kind() == reflect.Slice {
				var inType, finalType reflect.Type
				for i := 0; i < l; i++ {
					if i >= vmNumIn-1 { //more arguments
						inType = finalType
					} else {
						inType = vm.Type().In(i)
					}
					value, err := r.ObjectTransfer(margs[i])
					if err != nil {
						return nil, err
					}
					if value.IsZero() {
						value = reflect.New(inType).Elem()
					}
					finalType = value.Type()
					arguments = append(arguments, value)
				}
			} else {
				return nil, errors.New("function '" + name + "' has too many arguments")
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
			mret = []types.ExtendObject{}
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

	return types.NewMethod(name, vm, f), nil
}

func (r *Raev) newRawClass(source any, middlewares ...types.ClassMiddleware) (_ *types.Class, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = errors.New("Raev::RawClass::Panic@" + fmt.Sprint(rec))
		}
	}()
	//Prepare
	pGos := reflect.ValueOf(source)
	if pGos.IsZero() {
		pGos = reflect.New(reflect.TypeOf(source).Elem())
	}
	gos := pGos.Elem()
	pt := pGos.Type()
	t := pGos.Elem().Type()
	c := types.NewClass(pGos)

	//Set member vars
	for i := 0; i < gos.NumField(); i++ {
		vf := gos.Field(i)
		tf := t.Field(i)
		if !vf.CanInterface() {
			continue
		}
		if obj, err := r.ValueTransfer(vf.Interface()); err != nil {
			return nil, err
		} else {
			c.SetVar(tf.Name, types.NewArgument(obj, types.NewParameter(vf.Type())))
		}
	}

	//Set member methods
	for i := 0; i < pt.NumMethod(); i++ {
		mt := pt.Method(i)
		vm := pGos.Method(i)

		if method, err := r.NewRawMethod(mt.Name, vm); err == nil {
			c.SetMethod(mt.Name, method)
		} else {
			return nil, err
		}
	}

	//Handle Middlewares
	for _, m := range middlewares {
		m.Handle(&c)
	}

	return &c, nil
}

func (r *Raev) NewClass(name string, source any, middlewares ...types.ClassMiddleware) (_ types.ExtendClass, err error) {
	c, err := r.newRawClass(source, middlewares...)
	if err != nil {
		return nil, err
	}
	return r.trans.ToClass(name, c)
}

func NewRaev(zero types.ExtendObject, trans Transfer) *Raev {
	rv := &Raev{}
	rv.zeroObj = zero
	rv.trans = trans
	return rv
}
