package adz

var StdLib = make(map[string]Proc)

func init() {
	StdLib["proc"] = ProcProc
}

func ProcProc(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 4 {
		return EmptyToken, ErrArgCount(3, len(args)-1)
	}

	argList, err := args[2].AsList()
	if err != nil {
		return EmptyToken, ErrSyntax("could not parse proc argument list", err)
	}

	interp.Procs[args[1].String] = func(pinterp *Interp, pargs []*Token) (*Token, error) {
	again:
		if len(pargs)-1 != len(argList) && len(argList) > 1 && argList[len(argList)-1].String != "args" {
			return EmptyToken, ErrArgCount(pargs[0].String, len(argList), len(pargs)-1)
		}

		if pargs[0].String != "tailcall" {
			pinterp.Push()
			defer pinterp.Pop()
		}

		for i := range pargs[1:] {
			if argList[i].String == "args" {
				pinterp.SetVar("args", NewList(pargs[i+1:]))
				break
			}
			pinterp.SetVar(argList[i].String, pargs[i+1])
		}

		ret, err := pinterp.ExecToken(args[3])
		switch err {
		case ErrReturn:
			err = nil
		case ErrTailcall:
			pargs, _ = ret.AsList()
			goto again
		}

		return ret, err
	}

	return EmptyToken, nil
}

/*
	Given a list of args,

	AssignArgs
func AssignArgs(interp *Interp, argList []*Token, args []*Token) error {
	// each element of argList can be a one or two element list
	// An underscore means do not assign.

	for i := range argList {
		param, err := argList[i].AsList()
		if err != nil || len(param) > 2 {
			return err
		}


	}

	for i:= range args {
		if args[i].String[0] == '-' {
			// arg is a named arg, find a matching name in argList
			for j := range argList {
				if argList[j].String == args[i].String {
					interp.SetVar(arg
				}
			}
		}
	}
}
*/

func ParseArgs(args []*Token) (namedArgs map[string]*Token, posArgs []*Token, err error) {
	namedArgs = make(map[string]*Token)
	for i := 0; i < len(args); i++ {
		if len(args[i].String) <= 1 {
			posArgs = append(posArgs, args[i])
			continue
		}
		if args[i].String[0] == '-' {
			// either a named arg or the '--' terminator
			if args[i].String[1] == '-' {
				// got the -- terminator, append remaining args to posArgs
				posArgs = append(posArgs, args[i+1:]...)
				return
			}

			if i+1 >= len(args) {
				err = ErrNamedArgMissingValue(args[i].String)
				return
			}

			// otherwise, set a named arg
			namedArgs[args[i].String[1:]] = args[i+1]
			i++
			continue
		}

		posArgs = append(posArgs, args[i])
	}

	return
}
