package types

type Argument struct {
	*Parameter
	v ExtendObject
}

func (a *Argument) Object() ExtendObject {
	return a.v
}

func NewArgument(v ExtendObject, p Parameter) Argument {
	return Argument{
		Parameter: &p,
		v:         v,
	}
}
