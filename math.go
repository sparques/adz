package adz

func init() {
	StdLib["eq"] = ProcEq
	StdLib["=="] = ProcEq
	StdLib["ne"] = ProcNeq
	StdLib["!="] = ProcNeq
	StdLib["not"] = ProcNot
	StdLib["and"] = ProcAnd
	StdLib["or"] = ProcOr
	StdLib["sum"] = ProcSum
	StdLib["+"] = ProcSum
	StdLib["lt"] = procDiadicCmp(lessThan[int])
	StdLib["<"] = procDiadicCmp(lessThan[int])
	StdLib["lte"] = procDiadicCmp(lessThanOrEqual[int])
	StdLib["<="] = procDiadicCmp(lessThanOrEqual[int])
	StdLib["gt"] = procDiadicCmp(lessThan[int])
	StdLib[">"] = procDiadicCmp(lessThan[int])
	StdLib["gte"] = procDiadicCmp(greaterThanOrEqual[int])
	StdLib[">="] = procDiadicCmp(greaterThanOrEqual[int])
}

// ProcEq performs a shallow comparison
func ProcEq(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 2; i < len(args); i++ {
		if args[1].String != args[2].String {
			return FalseToken, nil
		}
	}

	return TrueToken, nil
}

// ProcNeq performs a shallow comparison
func ProcNeq(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(args[0], 1, len(args)-1)
	}

	if args[1].String == args[2].String {
		return TrueToken, nil
	}

	return FalseToken, nil
}

// ProcNot performs a boolean not
func ProcNot(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(args[0], 1, len(args)-1)
	}

	b, err := args[1].AsBool()
	if err != nil {
		return EmptyToken, ErrExpectedBool(args[1].String)
	}

	if b {
		return FalseToken, nil
	}

	return TrueToken, nil
}

func ProcAnd(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 1; i < len(args); i++ {
		v, err := args[i].AsBool()
		if err != nil {
			return EmptyToken, ErrExpectedBool(args[0], i-1, args[i])
		}
		if !v {
			return FalseToken, nil
		}
	}

	return TrueToken, nil
}

func ProcOr(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 1; i < len(args); i++ {
		v, err := args[i].AsBool()
		if err != nil {
			return EmptyToken, ErrExpectedBool(args[0], i-1, args[i])
		}
		if v {
			return TrueToken, nil
		}
	}

	return FalseToken, nil
}

// ProcSum
func ProcSum(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}
	var tot int
	for i := 1; i < len(args); i++ {
		j, err := args[i].AsInt()
		if err != nil {
			return EmptyToken, ErrExpectedInt(args[i].String)
		}
		tot += j
	}

	return NewTokenInt(tot), nil
}

// ProcDiff

// ProcMult

// ProcDiv

// ProcIncr

// Proc

// procDiadic lets you make a diadic Adz Proc from a simpler diadic golang func
func procDiadic(fn func(int, int) int) Proc {
	return func(interp *Interp, args []*Token) (*Token, error) {
		if len(args) != 3 {
			return EmptyToken, ErrArgCount(2, len(args)-1)
		}

		a, err := args[1].AsInt()
		if err != nil {
			return EmptyToken, ErrExpectedInt(args[1].String)
		}
		b, err := args[2].AsInt()
		if err != nil {
			return EmptyToken, ErrExpectedInt(args[1].String)
		}

		return NewToken(fn(a, b)), nil
	}
}

// procDiadic lets you make a diadic compare Adz Proc from a simpler diadic golang func
func procDiadicCmp(fn func(int, int) bool) Proc {
	return func(interp *Interp, args []*Token) (*Token, error) {
		if len(args) != 3 {
			return EmptyToken, ErrArgCount(2, len(args)-1)
		}

		a, err := args[1].AsInt()
		if err != nil {
			return EmptyToken, ErrExpectedInt(args[1].String)
		}
		b, err := args[2].AsInt()
		if err != nil {
			return EmptyToken, ErrExpectedInt(args[1].String)
		}

		return NewToken(fn(a, b)), nil
	}
}

func lessThan[N Number](a, b N) bool {
	return a < b
}

func lessThanOrEqual[N Number](a, b N) bool {
	return a <= b
}

func greaterThan[N Number](a, b N) bool {
	return a > b
}

func greaterThanOrEqual[N Number](a, b N) bool {
	return a >= b
}
