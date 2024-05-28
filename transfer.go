package raev

import "github.com/B9O2/raev/types"

type Transfer interface {
	ToObject(any) (types.ExtendObject, error)
	MakeSlice() types.ExtendSlice
	AppendSlice(types.ExtendSlice, types.ExtendObject) types.ExtendSlice
	MakeMap() types.ExtendMap
	SetMap(types.ExtendMap, types.ExtendObject, types.ExtendObject) (types.ExtendMap, error)
	ToValue(types.ExtendObject) (any, error)
	ToClass(string, *types.Class) (types.ExtendClass, error)
	ToMethod(types.Method) (types.ExtendMethod, error)
}

type BaseTransfer struct {
}

func (bt *BaseTransfer) MakeSlice() types.ExtendSlice {
	return nil
}

func (bt *BaseTransfer) AppendSlice(types.ExtendSlice, types.ExtendObject) types.ExtendSlice {
	return nil
}

func (bt *BaseTransfer) MakeMap() types.ExtendMap {
	return nil
}

func (bt *BaseTransfer) SetMap(types.ExtendMap, types.ExtendObject, types.ExtendObject) (types.ExtendMap, error) {
	return nil, nil
}
