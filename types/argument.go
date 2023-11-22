package types

type Argument struct {
	*Parameter
	v ExtendObject
}

func NewArgument(v ExtendObject, p Parameter) Argument {
	return Argument{
		Parameter: &p,
		v:         v,
	}
}
