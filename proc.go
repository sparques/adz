package adz

import (
	"fmt"
	"maps"
	"strings"
)

var StdLib = make(map[string]Proc)

func init() {
	StdLib["proc"] = ProcProc
	StdLib["macro"] = ProcMacro
	StdLib["trace"] = ProcTrace
}

// do we want to have macros support arguments? if we do that then it's perhaps too similar
// to a proc? It's just a proc that doesn't isolate its vars
func ProcMacro(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	var (
		ns   *Namespace
		id   string
		name string = args[1].String
	)
	if strings.HasPrefix(name, "::") {
		ns, id, _ = interp.ResolveIdentifier(name, true)
	} else {
		ns, id = interp.Frame.localNamespace, name
	}

	ns.Procs[id] = func(pinterp *Interp, pargs []*Token) (*Token, error) {
		return pinterp.ExecToken(args[2])
	}

	return args[1], nil
}

func ProcProc(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 4 {
		return EmptyToken, ErrArgCount(3, len(args)-1)
	}

	namedProto, posProto, err := ParseProto(args[2])
	if err != nil {
		return EmptyToken, err
	}

	var (
		ns *Namespace
		id string
	)

	// if defined fully qualified, pluck out the namespace
	// otherwise just set the proc's home namespace to the local namespace
	name := args[1].String
	if strings.HasPrefix(name, "::") {
		ns, id, _ = interp.ResolveIdentifier(name, true)
	} else {
		ns, id = interp.Frame.localNamespace, name
	}

	procPath := ns.Qualified(id)

	proc := func(pinterp *Interp, pargs []*Token) (*Token, error) {
		// check and set 'assume' named values here. if namedProto has a match in the
		// $use variable (lol, another todo: implement hashmaps), update its default value
		// to be the same as what's specified in $use
		parsedArgs, err := ParseArgs(namedProto, posProto, pargs[1:])
		if err != nil {
			return EmptyToken, err
		}

		if pargs[0].String != "tailcall" {
			pinterp.Push(&Frame{
				localVars:      parsedArgs,
				localNamespace: ns,
			})
			defer pinterp.Pop()
		}
	again:

		ret, err := pinterp.ExecToken(args[3])
		switch err {
		case ErrReturn:
			err = nil
		case ErrTailcall:
			pargs, _ = ret.AsList()
			reParsedArgs, err := ParseArgs(namedProto, posProto, pargs[1:])
			if err != nil {
				return EmptyToken, err
			}
			// TODO: clear parsed args before copying reParsedArgs
			//parsedArgs =
			maps.Copy(parsedArgs, reParsedArgs)
			goto again
		}

		return ret, err
	}

	ns.Procs[id] = proc
	tok := NewTokenString(procPath)
	tok.Data = proc
	return tok, nil
}

// ParseProto parses a proc argument prototype, returning the list of named args
// and the list of positional args.
func ParseProto(proto *Token) (namedProto []*Token, posProto []*Token, err error) {
	protoList, err := proto.AsList()
	if err != nil {
		return nil, nil, err
	}
	namedProto = []*Token{}
	posProto = []*Token{}
	for i := range protoList {
		switch {
		case len(protoList[i].Index(0).String) < 2:
			posProto = append(posProto, protoList[i])
		case protoList[i].Index(0).String[0] == '-':
			namedProto = append(namedProto, protoList[i])
		default:
			posProto = append(posProto, protoList[i])
		}
	}

	return
}

func isVariadic(l []*Token) bool {
	for i := range l {
		if l[i].Index(0).String == "-args" || l[i].String == "args" {
			return true
		}
	}
	return false
}

func protoContains(proto []*Token, e *Token) bool {
	for i := range proto {
		if proto[i].Index(0).String == e.String {
			return true
		}
	}
	return false
}

// consider...
//func AssignNamedArgs(parsedArgs map[string]*Token, namedProto []*Token, args []*Token) (error)
//func AssignPosArgs(parsedArgs map[string]*Token, posProto []*Token, args []*Token) (error)

func ParseArgs(namedProto []*Token, posProto []*Token, args []*Token) (parsedArgs map[string]*Token, err error) {
	parsedArgs = make(map[string]*Token)
	// shortcut for no-args given, no args accepted
	if len(namedProto) == 0 && len(posProto) == 0 && len(args) == 0 {
		return
	}

	posArgs := []*Token{}

	allNames := isVariadic(namedProto)
	processNamed := true
	for i := 0; i < len(args); i++ {
		if len(args[i].String) < 2 {
			posArgs = append(posArgs, args[i])
			continue
		}
		if processNamed {
			if args[i].String == "--" {
				processNamed = false
				continue
			}

			if args[i].String[0] == '-' {
				if i+1 >= len(args) {
					err = ErrExpectedMore
					return
				}
				// check if we even need this
				if !allNames && !protoContains(namedProto, args[i]) {
					err = ErrArgExtra(args[i].String)
					return
				}
				parsedArgs[args[i].String[1:]] = args[i+1]
				i++
				continue
			}
		}

		posArgs = append(posArgs, args[i])
	}

	// named args have been set and positional args placed into posArgs
	// assign pos args

	variadic := isVariadic(posProto)

	for i := range posProto {
		if i >= len(posArgs) {
			// we have more positional prototypes than args, attempt to set using default values
			if posProto[i].Len() == 2 {
				parsedArgs[posProto[i].Index(0).String] = posProto[i].Index(1)
				continue
			}

			// non-variadic is easy-- if the counts don't match, it's an error
			if !variadic {
				err = ErrArgCount(len(posProto), len(posArgs))
				return
			}

			if len(posArgs) < len(posProto)-1 {
				err = ErrArgMinimum(len(posProto)-1, len(posArgs))
				return
			}

		}

		// a posProto of "args" means cram all remaining args into args
		if posProto[i].Index(0).String == "args" {
			parsedArgs["args"] = NewList(posArgs[i:])
			break
		}

		parsedArgs[posProto[i].Index(0).String] = posArgs[i]
	}

	// check if we were passed more positional args than we have positional prototypes
	// but only if we're not variadic
	if !variadic {
		if len(posArgs) > len(posProto) {
			err = ErrArgCount(len(posProto), len(posArgs))
			return
		}
	}

	// set any named args to their default value if they weren't set
	for i := range namedProto {
		if namedProto[i].String == "-args" {
			continue
		}
		if _, ok := parsedArgs[namedProto[i].Index(0).String[1:]]; !ok {
			if namedProto[i].Len() == 2 {
				parsedArgs[namedProto[i].Index(0).String[1:]] = namedProto[i].Index(1)
				continue
			}

			err = ErrArgMissing(namedProto[i].Index(0).String)
			return
		}
	}

	return
}

func ParseArgsWithProto(prototype string, args []*Token) (map[string]*Token, error) {
	namedProto, posProto, err := ParseProto(NewTokenString(prototype))
	if err != nil {
		return nil, err
	}
	return ParseArgs(namedProto, posProto, args)
}

func ProcTrace(interp *Interp, args []*Token) (*Token, error) {
	namedProto, posProto, _ := ParseProto(NewTokenString(`varName traceProcName`))
	parsedArgs, err := ParseArgs(namedProto, posProto, args[1:])
	if err != nil {
		return EmptyToken, err
	}
	// resolve variable name
	ns, id, err := interp.ResolveIdentifier(parsedArgs["varName"].String, true)
	varName := ns.Qualified(id)

	// resolve proc name
	proc, err := interp.ResolveProc(parsedArgs["traceProcName"].String)
	if err != nil {
		return EmptyToken, fmt.Errorf("could not find proc %s: %w", parsedArgs["traceProcName"].String, err)
	}

	// setup the trace
	interp.Traces[varName] = proc

	// should return something else? Name of proc? :shrug:
	return EmptyToken, nil
}
