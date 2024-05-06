package adz

import (
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
}

// Command is a set of tokens.
type Command []*Token

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

func (interp *Interp) Push() {
	interp.Stack = append(interp.Stack, interp.Vars)
	interp.Vars = make(map[string]*Token)
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

func (interp *Interp) GetVar(name string) (*Token, error) {
	if tok, ok := interp.Vars[name]; ok {
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", name)
}

func (interp *Interp) SetVar(name string, val *Token) (*Token, error) {
	interp.Vars[name] = val
	return val, nil
}

func (interp *Interp) Exec(cmd Command) (*Token, error) {
	// Lex functions should skip empty commands
	// if len(cmd) == 0 {
	// 	return EmptyToken, nil
	// }

	// substitution pass
	var err error
	for i, tok := range cmd {
		cmd[i], err = interp.Subst(tok)
		if err != nil {
			return EmptyToken, fmt.Errorf("%s: error substituting arg %d: %w", cmd.Summary(), i, err)
		}
	}

	// proc look up
	if proc, ok := interp.Procs[cmd[0].String]; ok {
		return proc(interp, cmd)
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

	return EmptyToken, fmt.Errorf("command not found: %s", cmd[0].String)
}

func (interp *Interp) ExecScript(script Script) (ret *Token, err error) {
	for _, cmd := range script {
		ret, err = interp.Exec(cmd)
		if err != nil {
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

// Script -> []Commands -> Tokens -> interp Substitution -> run command

// EvalToken interprets a token as script and runs it.
/*
func (interp *Interp) EvalToken(token *Token) (*Token, error) {
	script, err := token.AsScript()
	if err != nil
	return interp.Run(script)
}
*/

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')
}