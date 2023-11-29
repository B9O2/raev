package raev

import (
	"errors"
	"fmt"
	"github.com/B9O2/raev/types"
	"reflect"
)

type Transfer interface {
	ToObject(any) (types.ExtendObject, error)
	ToValue(types.ExtendObject) (any, error)
	ToClass(string, *types.Class) (types.ExtendClass, error)
	ToMethod(types.Method) (types.ExtendMethod, error)
}

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

	if class, err := r.newRawClass(value); err != nil {
		return r.trans.ToObject(value)
	} else {
		return r.trans.ToObject(class)
	}

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

func (r *Raev) ObjectTransfer(obj types.ExtendObject, expected reflect.Type) (reflect.Value, error) {
	transfer, err := r.trans.ToValue(obj)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	if transfer != nil {
		return reflect.ValueOf(transfer), nil
	} else {
		return reflect.New(expected).Elem(), nil
	}
}

/*
// RegisterClassTransfer 注册类转换方法

	func (r *Raev) RegisterClassTransfer(classInit func(name string) (types.Class, error), classTransfer func(f func(kwargs map[string]types.ExtendObject) (types.Class, error)) (types.Method, error)) {
		r.cInit = classInit
		r.cTrans = classTransfer
	}
*/

func (r *Raev) NewMethod(name string, m any) (types.ExtendMethod, error) {
	method, err := r.newRawMethod(name, reflect.ValueOf(m))
	if err != nil {
		return nil, err
	}
	return r.trans.ToMethod(method)
}

func (r *Raev) newRawMethod(name string, vm reflect.Value) (types.Method, error) {
	f := func(margs []types.ExtendObject) (mret []types.ExtendObject, err error) {
		var arguments []reflect.Value
		mret = []types.ExtendObject{r.zeroObj}
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

func (r *Raev) newRawClass(source any) (_ *types.Class, err error) {
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

		if method, err := r.newRawMethod(mt.Name, vm); err == nil {
			c.SetMethod(mt.Name, method)
		} else {
			return nil, err
		}
	}
	return &c, nil
}

func (r *Raev) NewClass(name string, source any) (_ types.ExtendClass, err error) {
	c, err := r.newRawClass(source)
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
