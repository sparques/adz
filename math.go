package adz

func init() {
	StdLib["eq"] = ProcEq
	StdLib["ne"] = ProcNeq
	StdLib["not"] = ProcNot
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
		return EmptyToken, ErrExpectedBool(args[0], 1, args[1])
	}

	if b {
		return FalseToken, nil
	}

	return TrueToken, nil
}

// ProcSum

// ProcDiff

// ProcMult

// ProcDiv

// ProcIncr

// Proc 