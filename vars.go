package adz

func init() {
	StdLib["set"] = ProcSet
	StdLib["del"] = ProcDel
	StdLib["subst"] = ProcSubst
	StdLib["var"] = ProcVar
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

func ProcSubst(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}
	return interp.Subst(args[1])
}

//  ProcIdx (equiv to lindex

// var varname - returns true/false if varname exists or doesn't exist
// var varname {val} sets varname to val
// var varname cmd <args> does a variable subcommand like...
// var varname idx n ;# treat varname as a list and return the nth index of varname;
// var varname idx {n1 n2 n3...} ;# treat varname as a list and return the n3-th index of the n2-th index of the n1-th index of varname;
// var varname idx n val;# treat varname as a list and set the nth index of varname to val
// var varname len ;# treat varname as a list and return its length
// var varname append ;# append
func ProcVar(interp *Interp, args []*Token) (*Token, error) {
	switch len(args) {
	case 1:
		// var command by itself, list out vars
		out := make([]*Token, 0, len(interp.Vars))
		for k, v := range interp.Vars {
			out = append(out, NewList([]*Token{NewTokenString(k), v}))
		}
		return NewList(out), nil
	case 2: // var <varname> ; return true/false if var exists
		if _, ok := interp.Vars[args[1].String]; ok {
			return TrueToken, nil
		}
		return FalseToken, nil
	default:
		vari, err := interp.GetVar(args[1].String)
		if err != nil {
			return EmptyToken, err
		}
		switch args[2].String {
		case "len":
			variList, err := vari.AsList()
			if err != nil {
				return NewTokenInt(1), nil
			}
			return NewTokenInt(len(variList)), nil
		case "idx":
			if len(args) != 4 {
				return EmptyToken, ErrArgCount(3, len(args)-1)
			}
			idxList, err := args[3].AsList()
			if err != nil {
				return EmptyToken, ErrSyntax
			}

			for _, idxTok := range idxList {
				idx, err := idxTok.AsInt()
				if err != nil {
					return EmptyToken, ErrExpectedInt
				}
				vari = vari.Index(idx)
				// if vari == EmptyToken {
				// 	break
				// }
			}

			return vari, nil

		}
	}

	return EmptyToken, nil

}
