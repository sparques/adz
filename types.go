package adz

import "golang.org/x/exp/constraints"

var (
	TrueToken  = &Token{String: "true", Data: true}
	FalseToken = &Token{String: "false", Data: false}
)

type Number interface {
	constraints.Integer | constraints.Float
}

func init() {
	StdLib["bool"] = ProcBool
	StdLib["int"] = ProcInt
	StdLib["float"] = ProcFloat
	StdLib["true"] = ProcTrue
	StdLib["false"] = ProcFalse
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
