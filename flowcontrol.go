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

type ifClause struct {
	cond *Token
	body *Token
}

func ProcIf(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum("if", 2, len(args)-1)
	}

	clauses := make([]ifClause, 0, 2)
	var elseBody *Token
	arg := 1

	parseBranch := func() error {
		if arg >= len(args) {
			return ErrExpectedMore("conditional expression", "if")
		}
		cond := args[arg]
		arg++
		if arg < len(args) && args[arg].String == "then" {
			arg++
		}
		if arg >= len(args) {
			return ErrExpectedMore("script body", "then")
		}
		clauses = append(clauses, ifClause{cond: cond, body: args[arg]})
		arg++
		return nil
	}

	if err := parseBranch(); err != nil {
		return EmptyToken, err
	}

	for arg < len(args) {
		switch args[arg].String {
		case "elseif":
			arg++
			if err := parseBranch(); err != nil {
				return EmptyToken, err
			}
		case "else":
			arg++
			if arg >= len(args) {
				return EmptyToken, ErrExpectedMore("script body", "else")
			}
			elseBody = args[arg]
			arg++
			if arg != len(args) {
				return EmptyToken, ErrSyntaxExpected("end of if", args[arg].String)
			}
		default:
			return EmptyToken, ErrSyntaxExpected("elseif or else", args[arg].String)
		}
	}

	for i, clause := range clauses {
		cond, err := interp.ExecToken(clause.cond)
		if err != nil {
			return EmptyToken, ErrEvalCond(i, err)
		}
		b, err := cond.AsBool()
		if err != nil {
			return EmptyToken, ErrEvalCond(i, err)
		}
		if b {
			return interp.ExecToken(clause.body)
		}
	}

	if elseBody != nil {
		return interp.ExecToken(elseBody)
	}

	return EmptyToken, nil
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
		return EmptyToken, ErrArgCount(4, len(args)-1)
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
	varList, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}
	if len(varList) == 0 {
		return EmptyToken, ErrArgMissing("variable list")
	}
	ret = EmptyToken
	for i := 0; i < len(list); i += len(varList) {
		// set vars...
		for j := range varList {
			if i+j >= len(list) {
				interp.SetVar(varList[j].String, EmptyToken)
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
	if len(args) == 4 && args[2].String != "while" {
		return EmptyToken, ErrSyntaxExpected("while", args[2].String)
	}

	var ret = EmptyToken
	var err error

	for {
		ret, err = interp.ExecToken(args[1])
		loopControl := err

		switch err {
		case nil:
		case ErrBreak:
			return ret, nil
		case ErrContinue:
			err = nil
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
		if loopControl == ErrContinue {
			continue
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
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum(1, len(args)-1)
	}
	if len(args) > 4 {
		return EmptyToken, ErrArgCount(3, len(args)-1)
	}

	ret, err := interp.ExecToken(args[1])

	if len(args) > 2 {
		interp.SetVar(args[2].String, ret)
	}

	if len(args) > 3 {
		errTok := EmptyToken
		if err != nil {
			errTok = NewToken(err)
		}
		interp.SetVar(args[3].String, errTok)
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
	if len(args) > 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}
	if len(args) == 2 {
		return args[1], ErrContinue
	}
	return EmptyToken, ErrContinue
}

func ProcBreak(interp *Interp, args []*Token) (*Token, error) {
	if len(args) > 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}
	if len(args) == 2 {
		return args[1], ErrBreak
	}
	return EmptyToken, ErrBreak
}

func ProcReturn(interp *Interp, args []*Token) (*Token, error) {
	if len(args) > 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}
	if len(args) == 2 {
		return args[1], ErrReturn
	}
	return EmptyToken, ErrReturn
}

func ProcTailcall(interp *Interp, args []*Token) (*Token, error) {
	return NewList(args), ErrTailcall
}
