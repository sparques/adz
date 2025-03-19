package adz

import (
	"errors"
	"fmt"
	"io"
	"maps"
)

type Runable interface {
	Exec() (*Token, error)
}

type Interp struct {
	Stdin  io.Reader
	Stdout io.Writer
	Procs  map[string]Proc
	Vars   map[string]*Token
	Stack  []map[string]*Token

	calldepth int
}

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
	return &Interp{
		Stdout: io.Discard,
		Stdin:  &NilReader{},
		Procs:  maps.Clone(StdLib),
		Vars:   make(map[string]*Token),
	}
}

func (interp *Interp) Push(newEnv ...map[string]*Token) {
	interp.Stack = append(interp.Stack, interp.Vars)
	if len(newEnv) == 1 {
		interp.Vars = newEnv[0]
	} else {
		interp.Vars = make(map[string]*Token)
	}
}

func (interp *Interp) Pop() {
	if len(interp.Stack) == 0 {
		return
	}
	interp.Vars = interp.Stack[len(interp.Stack)-1]
	interp.Stack = interp.Stack[:len(interp.Stack)-1]
}

func (interp *Interp) Proc(name string, proc Proc) {
	if proc == nil {
		delete(interp.Procs, name)
		return
	}
	interp.Procs[name] = proc
}

func (interp *Interp) LoadProcs(procset map[string]Proc) {
	maps.Copy(interp.Procs, procset)
}

func (interp *Interp) GetVar(name string) (*Token, error) {
	if tok, ok := interp.Vars[name]; ok {
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", name)
}

// ResolveVar checks current scope and all parent scopes for a variable.
func (interp *Interp) ResolveVar(name string) (*Token, error) {
	if tok, ok := interp.Vars[name]; ok {
		return tok, nil
	}
	for i := len(interp.Stack) - 1; i >= 0; i-- {
		if tok, ok := interp.Stack[i][name]; ok {
			return tok, nil
		}
	}
	return EmptyToken, fmt.Errorf("no such variable %s", name)
}

func (interp *Interp) SetVar(name string, val *Token) (*Token, error) {
	interp.Vars[name] = val
	return val, nil
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

	// proc look up
	if proc, ok := interp.Procs[args[0].String]; ok {
		//return proc(interp, args)
		ret, err := proc(interp, args)
		if err != nil && !errors.Is(err, ErrFlowControl) {
			err = ErrCommand(args[0].String, err)
		}
		return ret, err
	}

	// proc wasn't found, check if empty string proc exists and call that if it does
	if unknown, ok := interp.Procs[""]; ok {
		ret, err := unknown(interp, args)
		if err != nil && !errors.Is(err, ErrFlowControl) {
			err = ErrCommand(args[0].String, err)
		}
		return ret, err
	}

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
