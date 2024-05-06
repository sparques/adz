package adz

func init() {
	StdLib["set"] = ProcSet
	StdLib["del"] = ProcDel
}

func ProcSet(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	return interp.SetVar(args[1].String, args[2])
}

func ProcDel(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum(1, len(args)-1)
	}

	for _, tok := range args[1:] {
		_, ok := interp.Vars[tok.String]
		if !ok {
			return EmptyToken, ErrNoVar(tok.String)
		}

		delete(interp.Vars, tok.String)
	}

	return EmptyToken, nil
}

// ProcList

//  ProcIdx (equiv to lindex
