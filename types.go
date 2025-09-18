package adz

import "golang.org/x/exp/constraints"

var (
	TrueToken  = &Token{String: "true", Data: true}
	FalseToken = &Token{String: "false", Data: false}
)

type Number interface {
	constraints.Integer | constraints.Float
}

type Integer interface {
	Int() int
}

// Floater is an interface so an otherwise opaque object can signal it is a
// a float value.
type Floater interface {
	Float() float64
}

// Float and it's single method Float() is a helper to easily pass float64s
// to things expecting a Floater interface.
type Float float64

func (f Float) Float() float64 {
	return float64(f)
}

func init() {
	StdLib["bool"] = ProcBool
	StdLib["int"] = ProcInt
	StdLib["float"] = ProcFloat
	StdLib["true"] = ProcTrue
	StdLib["false"] = ProcFalse
	StdLib["tuple"] = ProcTuple
}

func ProcBool(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	_, err := args[1].AsBool()
	return args[1], err
}

func ProcInt(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 1; i < len(args); i++ {
		_, err := args[i].AsInt()
		if err != nil {
			return EmptyToken, err
		}
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return NewList(args[1:]), nil
}

func ProcFloat(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 1; i < len(args); i++ {
		_, err := args[i].AsFloat()
		if err != nil {
			return EmptyToken, err
		}
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return NewList(args[1:]), nil
}

func ProcTrue(interp *Interp, args []*Token) (*Token, error) {
	return TrueToken, nil
}

func ProcFalse(interp *Interp, args []*Token) (*Token, error) {
	return FalseToken, nil
}

func ProcTuple(interp *Interp, args []*Token) (*Token, error) {
	// TODO: add fancy arg parsing and suport for {-matchcase true bool {Match case. If false, return value is normalized to all lower case}}
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2)
	}

	list, _ := args[1].AsList()

	return args[2].AsTuple(list)
}
