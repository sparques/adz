package adz

func init() {
	StdLib["if"] = ProcIf
	StdLib["while"] = ProcWhile
	StdLib["break"] = ProcBreak
	StdLib["return"] = ProcReturn
	StdLib["continue"] = ProcContinue
	StdLib["tailcall"] = ProcTailcall
}

func ProcIf(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum("if", 2, len(args)-1)
	}

	arg := 1
	for {
		cond, err := interp.ExecToken(args[arg])
		if err != nil {
			return EmptyToken, err
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, err
		}
		arg++
		if arg >= len(args) {
			return EmptyToken, ErrSyntax
		}
		if args[arg].String == "then" {
			arg++
			if arg >= len(args) {
				return EmptyToken, ErrSyntax
			}
		}
		if b {
			return interp.ExecToken(args[arg])
		}
		// condition wasn't true; if there's no more args, we can just return without error
		arg++
		if arg >= len(args) {
			return EmptyToken, nil
		}
		if args[arg].String == "elseif" {
			arg++
			if arg >= len(args) {
				return nil, ErrSyntax
			}
			continue
		}
		if args[arg].String == "else" {
			arg++
			if arg >= len(args) {
				return EmptyToken, ErrSyntax
			}
			return interp.ExecToken(args[arg])
		}
	}
}

func ProcWhile(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	var ret = EmptyToken

	for {
		cond, err := interp.ExecToken(args[1])
		if err != nil {
			return EmptyToken, ErrEvalCond("while", err)
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, ErrCondNotBool("while", cond.String)
		}

		if !b {
			return ret, nil
		}

		ret, err = interp.ExecToken(args[2])
		switch err {
		case nil:
		case ErrBreak:
			return ret, nil
		case ErrContinue:
			continue
		default:
			return ret, err
		}
	}
}

// ProcFor

// ProcForeach

// ProcDoWhile

// ProcCatch

func ProcContinue(interp *Interp, args []*Token) (*Token, error) {
	return EmptyToken, ErrContinue
}

func ProcBreak(interp *Interp, args []*Token) (*Token, error) {
	if len(args) == 2 {
		return args[1], ErrBreak
	}
	return EmptyToken, ErrBreak
}

func ProcReturn(interp *Interp, args []*Token) (*Token, error) {
	if len(args) == 2 {
		return args[1], ErrReturn
	}
	return EmptyToken, ErrReturn
}

func ProcTailcall(interp *Interp, args []*Token) (*Token, error) {
	return NewList(args), ErrTailcall
}
