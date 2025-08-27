package adz

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
)

type Runable interface {
	Exec() (*Token, error)
}

type Interp struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Namespaces map[string]*Namespace
	// Current (executing) namespace
	Namespace *Namespace

	Stack []*Frame
	Frame *Frame

	// Traces is a map of variables. When a variable is called/used, its fully
	// qualified name is checked in Traces. If a proc exists it is executed
	// with args[0] being the variable itself, and arg[1] being the action:
	//get, set, or del. The return value of this proc is what is returned when the action is get or del. When the action is set, argv[2] will have the to-be-
	// assigned value.
	Traces map[string]Proc

	calldepth int
}

type Frame struct {
	localNamespace *Namespace
	localVars      map[string]*Token
}

// alternatively

// Command is just a list of tokens.
type Command = List

func (cmd Command) Summary() string {
	out := ""
	for _, arg := range cmd {
		out += " " + arg.Summary()
	}
	return out[1:]
}

// Script is a set of a commands and a set of commands are a set of tokens
type Script []Command

type Proc func(*Interp, []*Token) (*Token, error)

type NilReader struct{}

// should always return 0, EOF instead of 0, nil ?
func (nr *NilReader) Read([]byte) (n int, err error) {
	return 0, nil
}

func NewInterp() *Interp {
	globalns := NewNamespace("")
	globalns.Procs = maps.Clone(StdLib)
	nses := make(map[string]*Namespace)
	nses[""] = globalns

	interp := &Interp{
		Stdout:     io.Discard,
		Stdin:      &NilReader{},
		Namespaces: nses,
		Namespace:  globalns,
		Stack:      []*Frame{},
		Frame: &Frame{
			localNamespace: globalns,
			localVars:      globalns.Vars,
		},
		Traces: make(map[string]Proc),
	}
	return interp
}

func (interp *Interp) Push(frame *Frame) {
	interp.Stack = append(interp.Stack, interp.Frame)
	interp.Frame = frame
}

func (interp *Interp) Pop() {
	if len(interp.Stack) == 0 {
		return
	}
	interp.Frame = interp.Stack[len(interp.Stack)-1]
	interp.Stack = interp.Stack[:len(interp.Stack)-1]
}

func (interp *Interp) Proc(name string, proc Proc) (err error) {
	if proc == nil {
		ns, id, err := interp.ResolveIdentifier(name, false)
		if err != nil {
			return err
		}
		delete(ns.Procs, id)
		return nil
	}
	ns, id, err := interp.ResolveIdentifier(name, true)
	ns.Procs[id] = proc
	return nil
}

func (interp *Interp) LoadProcs(ns *Namespace, procset map[string]Proc) {
	maps.Copy(ns.Procs, procset)
}

func (interp *Interp) ResolveProc(name string) (Proc, error) {
	// if it is a fully qualified id, we can skip to a look up
	if strings.HasPrefix(name, "::") {
		ns, id, err := interp.ResolveIdentifier(name, false)
		if err != nil {
			return nil, err
		}
		return ns.Procs[id], nil
	}

	// relative path given, step through our search order.
	// 1: check home namespace
	proc, ok := interp.Frame.localNamespace.Procs[name]
	if ok {
		return proc, nil
	}
	// 2: check currently executing namespace
	proc, ok = interp.Namespace.Procs[name]
	if ok {
		return proc, nil
	}
	// 3: final attempt, global namespace
	proc, ok = interp.Namespaces[""].Procs[name]
	if ok {
		return proc, nil
	}

	return nil, ErrCommandNotFound
}

// AbsoluteProc exclusively takes a fully qualified path and returns the matching
// proc if found. Otherwise it returns nil.
func (interp *Interp) AbsoluteProc(qualPath string) Proc {
	if !strings.HasPrefix(qualPath, "::") {
		return nil
	}
	ns, id, _ := interp.ResolveIdentifier(qualPath, false)
	if ns == nil {
		return nil
	}

	return ns.Procs[id]
}

// ResolveVar checks current scope and all parent scopes for a variable.
func (interp *Interp) ResolveVar(name string) (*Token, error) {
	ns, id, err := interp.ResolveIdentifier(name, false)
	if err != nil {
		return EmptyToken, err
	}

	if tok, ok := ns.Vars[id]; ok {
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", name)
}

func (interp *Interp) GetVar(name string) (v *Token, err error) {
	if strings.HasPrefix(name, "::") {
		// already have fully qualified name, just use getVar
		return interp.getVar(name)
	}

	// must do hierachial look up.

	tok, ok := interp.Frame.localVars[name]
	if ok {
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", name)

	/*
	   // trace can work on non-existent variables, so do that first
	   ns, id, err := interp.ResolveIdentifier(name, false)

	   	if p, ok := interp.Traces[ns.Qualified(id)]; ok {
	   		varTok := EmptyToken
	   		// trace exists, run it
	   		if ns != nil {
	   			_, ok := ns.Vars[id]

	   			if ok {
	   				varTok = ns.Vars[id]
	   			}
	   		}
	   		rez, err := p(interp, []*Token{varTok, NewTokenString("get"), NewTokenString(name)})
	   		if errors.Is(err, ErrBreak) {
	   			// if err is ErrBreak, return rez rather than the actual variable value
	   			return rez, nil
	   		}
	   	}
	*/
}

func (interp *Interp) getVar(qualName string) (*Token, error) {
	ns, id, err := interp.ResolveIdentifier(qualName, false)
	if err != nil {
		return EmptyToken, err
	}
	if tok, ok := ns.Vars[id]; ok {
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", qualName)
}

func (interp *Interp) SetVar(name string, val *Token) (*Token, error) {
	if strings.HasPrefix(name, "::") {
		ns, id, err := interp.ResolveIdentifier(name, true)
		if err != nil {
			return EmptyToken, err
		}
		ns.Vars[id] = val
		return val, nil
	}
	// otherwise we're just setting localvar
	interp.Frame.localVars[name] = val
	return val, nil
}

func (interp *Interp) DelVar(name string) (*Token, error) {
	ns, id, err := interp.ResolveIdentifier(name, true)
	if err != nil {
		return EmptyToken, err
	}

	v, ok := ns.Vars[name]
	if !ok {
		return EmptyToken, ErrNoVar
	}
	delete(ns.Vars, id)
	return v, nil
}

func (interp *Interp) CallDepth() int {
	return interp.calldepth
}

func (interp *Interp) Exec(cmd Command) (*Token, error) {
	// Lex functions should skip empty commands
	// if len(cmd) == 0 {
	// 	return EmptyToken, nil
	// }
	interp.calldepth++
	defer func() { interp.calldepth-- }()

	// substitution pass
	var err error
	var args = make([]*Token, len(cmd))
	for i, tok := range cmd {
		args[i], err = interp.Subst(tok)
		if err != nil {
			if errors.Is(err, ErrFlowControl) {
				return EmptyToken, err
			}
			return EmptyToken, fmt.Errorf("%s: error substituting arg %d: %w", cmd[0], i, err)
		}
	}

	// special case; if the underlying type of the first arg is a Proc, run that Proc
	if proc, ok := args[0].Data.(Proc); ok {
		ret, err := proc(interp, args)
		if err != nil && !errors.Is(err, ErrFlowControl) {
			err = ErrCommand(args[0].String, err)
		}
		return ret, err
	}

	// proc look up
	proc, err := interp.ResolveProc(args[0].String)
	if err != nil || proc == nil {
		// no dice. Try an unknown proc
		proc, err = interp.ResolveProc("")
		if err != nil || proc == nil {
			return EmptyToken, ErrCommandNotFound(args[0].String)
		}
	}
	ret, err := proc(interp, args)
	if err != nil && !errors.Is(err, ErrFlowControl) {
		err = ErrCommand(args[0].String, err)
	}
	return ret, err

	/* fix this later
	// try parsing cmd[0] as a list. If we parse it as
	// a list successfully and it's a two element list,
	// try running it as an anonymous proc
	list, err := cmd[0].AsList()
	if err == nil && len(list) == 2 {
		// list[0] := arglist
		// list[1] := procbody
		argList, err := list[0].AsList()
		if err != nil {
			return EmptyToken, fmt.Errorf("couldn't parse assumed anonymous proc arglist: %s", err)
		}

		body, err := list[1].AsScript()
		if err != nil {
			return EmptyToken, fmt.Errorf("couldn't parse assumed anonymous proc body: %s", err)
		}

		interp.Push()
		defer interp.Pop()
		for i := range cmd[1:] {
			interp.SetVar(argList[i].String, cmd[i])
		}
		ret, err := interp.ExecScript(body)
		if err == ErrReturn {
			err = nil
		}
		return ret, err

	}
	*/

	return EmptyToken, ErrCommandNotFound(cmd[0].String)
}

func (interp *Interp) ExecScript(script Script) (ret *Token, err error) {
	ret = EmptyToken
	for line, cmd := range script {
		ret, err = interp.Exec(cmd)
		if err != nil {
			if !errors.Is(err, ErrFlowControl) && line != 0 {
				return ret, ErrLine(line, err)
			}
			return ret, err
		}
	}

	return ret, err
}

func (interp *Interp) ExecToken(tok *Token) (*Token, error) {
	// first check if token is already parsed as a Script or Command
	if len(tok.String) == 0 {
		return EmptyToken, nil
	}
	switch v := tok.Data.(type) {
	case Script:
		return interp.ExecScript(v)
	case Command:
		return interp.Exec(v)
	default:
		script, err := tok.AsScript()
		if err != nil {
			return EmptyToken, err
		}
		return interp.ExecScript(script)
	}
}

func (interp *Interp) ExecString(str string) (*Token, error) {
	// attempt to lex str as script
	script, err := LexString(str)
	if err != nil {
		return EmptyToken, err
	}
	return interp.ExecScript(script)
}

func (interp *Interp) Printf(format string, args ...any) {
	fmt.Fprintf(interp.Stdout, format, args...)
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')
}
