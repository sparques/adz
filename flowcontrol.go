package adz

func init() {
	StdLib["if"] = ProcIf
	StdLib["while"] = ProcWhile
	StdLib["do"] = ProcDoWhile
	StdLib["for"] = ProcFor
	StdLib["foreach"] = ProcForEach
	StdLib["break"] = ProcBreak
	StdLib["return"] = ProcReturn
	StdLib["continue"] = ProcContinue
	StdLib["tailcall"] = ProcTailcall
	StdLib["catch"] = ProcCatch
	StdLib["throw"] = ProcThrow
}

func ProcIf(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum("if", 2, len(args)-1)
	}

	arg := 1
	for {
		cond, err := interp.ExecToken(args[arg])
		if err != nil {
			return EmptyToken, ErrEvalCond(arg-1, err)
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, ErrEvalCond(arg-1, err)
		}
		arg++
		if arg >= len(args) {
			return EmptyToken, ErrSyntax
		}
		if args[arg].String == "then" {
			arg++
			if arg >= len(args) {
				return EmptyToken, ErrExpectedMore("script body", "then")
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
				return nil, ErrExpectedMore("conditional expression", "elseif")
			}
			continue
		}
		if args[arg].String == "else" {
			arg++
			if arg >= len(args) {
				return EmptyToken, ErrExpectedMore("script body", "else")
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
			return EmptyToken, ErrEvalCond(0, err)
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, ErrEvalCond(0, err)
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

// ProcFor for {initial} {cond} {step} {body}
func ProcFor(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 5 {
		return EmptyToken, ErrArgCount(5, len(args)-1)
	}

	var ret = EmptyToken

	// initial
	_, err := interp.ExecToken(args[1])
	if err != nil {
		return EmptyToken, ErrEvalBody(0, "initial", err)
	}

	for {
		cond, err := interp.ExecToken(args[2])
		if err != nil {
			return EmptyToken, ErrEvalCond(1, err)
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, ErrEvalCond(1, err)
		}

		if !b {
			return ret, nil
		}

		ret, err = interp.ExecToken(args[4])
		switch err {
		case nil, ErrContinue:
		case ErrBreak:
			return ret, nil
		default:
			return ret, ErrEvalBody("for", err)
		}

		_, err = interp.ExecToken(args[3])
		if err != nil {
			return EmptyToken, ErrEvalBody(2, "step", err)
		}
	}
}

// ProcForEach
func ProcForEach(interp *Interp, args []*Token) (ret *Token, err error) {
	if len(args) != 4 {
		return EmptyToken, ErrArgCount(3, len(args)-1)
	}

	list, err := args[2].AsList()
	if err != nil {
		return EmptyToken, err // ErrArg(2) ?
	}
	varList, _ := args[1].AsList()
	for i := 0; i < len(list); i += len(varList) {
		// set vars...
		for j := range varList {
			if i+j >= len(list) {
				interp.SetVar(varList[j].String, EmptyToken)
				i++
				continue
			}
			interp.SetVar(varList[j].String, list[i+j])
		}

		// eval vody
		ret, err = interp.ExecToken(args[3])
		switch err {
		case nil:
		case ErrContinue:
			err = nil
			continue
		case ErrBreak:
			return
		default:
			return
		}
	}

	return
}

// ProcDoWhile
func ProcDoWhile(interp *Interp, args []*Token) (*Token, error) {
	if !(len(args) == 4 || len(args) == 2) {
		return EmptyToken, ErrArgCount(4, len(args)-1)
	}

	var ret = EmptyToken
	var err error

	for {
		ret, err = interp.ExecToken(args[1])

		switch err {
		case nil:
		case ErrBreak:
			return ret, nil
		case ErrContinue:
			continue
		default:
			return ret, ErrEvalBody(0, "do", err)
		}

		if len(args) == 2 {
			return ret, err
		}

		cond, err := interp.ExecToken(args[3])
		if err != nil {
			return EmptyToken, ErrEvalCond(2, "while", err)
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, ErrEvalCond(2, "while", err)
		}

		if !b {
			return ret, nil
		}
	}
}

// ProcSwitch
// switch can optionally have a single argument so that the full statement
// switch ?-case false? ?-match <exact|glob|regex>? val { }
// switch ?-case false? ?-match <exact|glob|regex>? val case n {body1} case b
/*
	switch -case false -match glob $var {
		n* {
			body1
		}
		m {
			body2
		}
		... {
			body...
		}
		default {
		}
	}
*/

// ProcCatch
func ProcCatch(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 1 {
		return EmptyToken, ErrArgMinimum(1, 0)
	}

	ret, err := interp.ExecToken(args[1])

	if len(args) > 2 {
		interp.SetVar(args[2].String, ret)
	}

	if len(args) > 3 {
		interp.SetVar(args[3].String, NewToken(err))
	}

	if err == nil {
		return FalseToken, nil
	}

	return TrueToken, nil
}

// ProcThrow
func ProcThrow(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	return EmptyToken, args[1]
}

func ProcContinue(interp *Interp, args []*Token) (*Token, error) {
	if len(args) == 2 {
		return args[1], ErrContinue
	}
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
