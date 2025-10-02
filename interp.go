package adz

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"sync"
)

type Runable interface {
	Exec() (*Token, error)
}

type Interp struct {
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	Namespaces   map[string]*Namespace
	Stack        []*Frame
	Frame        *Frame
	Monotonic    Monotonic
	calldepth    int
	MaxCallDepth int

	// signal chan Signal

	*sync.Mutex
}

type Frame struct {
	localNamespace *Namespace
	localVars      map[string]*Token
	localProcs     map[string]Proc
	namespaceRoot  bool
}

func (f *Frame) Namespace() string {
	return f.localNamespace.Qualified("")
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

func (proc Proc) AsToken(str string) *Token {
	tok := NewToken(proc)
	tok.String = str
	return tok
}

// Proc proc? proc! Proc proc proc.
func (proc Proc) Proc(interp *Interp, args []*Token) (*Token, error) {
	return proc(interp, args)
}

type Monotonic map[string]uint

func (m Monotonic) Next(prefix string) string {
	_, ok := m[prefix]
	if ok {
		m[prefix]++
	} else {
		m[prefix] = 0
	}
	return fmt.Sprintf("%s#%d", prefix, m[prefix])
}

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
		Stdin:      &NilReader{},
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Namespaces: nses,
		Stack:      []*Frame{},
		Frame: &Frame{
			localNamespace: globalns,
			localVars:      globalns.Vars,
			localProcs:     globalns.Procs,
			namespaceRoot:  true,
		},
		Monotonic:    make(Monotonic),
		MaxCallDepth: 1024,
		Mutex:        &sync.Mutex{},
	}
	// standard library stuff
	interp.LoadProcs("list", ListLib)
	interp.LoadProcs("str", StringsProcs)
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

func (interp *Interp) LoadProcs(ns string, procset map[string]Proc) {
	if _, ok := interp.Namespaces[ns]; !ok {
		interp.Namespaces[ns] = NewNamespace(ns)
	}
	maps.Copy(interp.Namespaces[ns].Procs, procset)
}

func (interp *Interp) ResolveProc(name string) (Proc, error) {
	// if it is a fully qualified id, we can skip to a look up
	if isQualified(name) {
		proc := interp.AbsoluteProc(name)
		if proc == nil {
			return nil, ErrCommandNotFound
		}
		return proc, nil
	}

	// relative path given, step through our search order.
	// 1: check localProcs
	proc, ok := interp.Frame.localProcs[name]
	if ok {
		return proc, nil
	}
	// 2: check home namespace
	proc, ok = interp.Frame.localNamespace.Procs[name]
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
	if !isQualified(qualPath) {
		return nil
	}
	ns, id, _ := interp.ResolveIdentifier(qualPath, false)
	if ns == nil {
		return nil
	}

	return ns.Procs[id]
}

func (interp *Interp) GetVar(name string) (v *Token, err error) {
	if isQualified(name) {
		// already have fully qualified name, just use getVar
		return interp.getVar(name)
	}

	// relative name means localVars only
	tok, ok := interp.Frame.localVars[name]
	if ok {
		// if Data is a Getter, use that instead
		if get, ok := tok.Data.(Getter); ok {
			return get.Get(tok)
		}
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", name)
}

func (interp *Interp) getVar(qualName string) (*Token, error) {
	if !isQualified(qualName) {
		return EmptyToken, fmt.Errorf("identifier is not fully-qualified ")
	}
	ns, id, err := interp.ResolveIdentifier(qualName, false)
	if err != nil {
		return EmptyToken, err
	}
	if tok, ok := ns.Vars[id]; ok {
		// if Data is a Getter, use that instead
		if get, ok := tok.Data.(Getter); ok {
			return get.Get(tok)
		}
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", qualName)
}

func (interp *Interp) setVar(qualName string, tok *Token) (*Token, error) {
	if !isQualified(qualName) {
		return EmptyToken, fmt.Errorf("identifier is not fully-qualified ")
	}
	ns, id, err := interp.ResolveIdentifier(qualName, true)
	if err != nil {
		return EmptyToken, err
	}
	if tok, ok := ns.Vars[id]; ok {
		return tok, nil
	}
	return EmptyToken, fmt.Errorf("no such variable %s", qualName)
}

func (interp *Interp) SetVar(name string, val *Token) (*Token, error) {
	if isQualified(name) {
		ns, id, err := interp.ResolveIdentifier(name, true)
		if err != nil {
			return EmptyToken, err
		}
		if tok, ok := ns.Vars[id]; ok {
			if setter, ok := tok.Data.(Setter); ok {
				return setter.Set(tok, val)
			}
		}
		ns.Vars[id] = val
		return val, nil
	}

	// exception: assigning to a single underscore (_) doesn't actually assign
	if name == "_" {
		return val, nil
	}

	// otherwise we're just setting localvar
	if tok, ok := interp.Frame.localVars[name]; ok {
		if setter, ok := tok.Data.(Setter); ok {
			return setter.Set(tok, val)
		}
	}

	interp.Frame.localVars[name] = val
	return val, nil
}

func (interp *Interp) DelVar(name string) (*Token, error) {
	if isQualified(name) {
		ns, id, err := interp.ResolveIdentifier(name, true)
		if err != nil {
			return EmptyToken, err
		}
		if tok, ok := ns.Vars[id]; ok {
			if deleter, ok := tok.Data.(Deleter); ok {
				deleter.Del(tok)
			}
			delete(ns.Vars, id)
			return tok, nil
		}
	}
	// otherwise we're just setting localvar
	if tok, ok := interp.Frame.localVars[name]; ok {
		if deleter, ok := tok.Data.(Deleter); ok {
			deleter.Del(tok)
		}
		delete(interp.Frame.localVars, name)
		return tok, nil
	}
	return EmptyToken, ErrNoVar
}

func (interp *Interp) CallDepth() int {
	return interp.calldepth
}

func (interp *Interp) getProc(cmd *Token) (proc Proc, ok bool) {
	var err error

	// The Proc type has a method ensuring that all Procs
	// implement the Procer interface.
	if pr, ok := cmd.Data.(Procer); ok {
		return pr.Proc, ok
	}

	// regular proc handling
	proc, err = interp.ResolveProc(cmd.String)
	if err == nil && proc != nil {
		return proc, true
	}

	// couldn't locate the given proc, check if there's an unknown/empty
	// proc to run.
	proc, err = interp.ResolveProc("")
	if err == nil && proc != nil {
		return proc, true
	}

	return nil, false
}

// ExecLiteral executes cmd without first doing a substitution pass.
func (interp *Interp) ExecLiteral(cmd Command) (tok *Token, err error) {
	// get proc
	proc, found := interp.getProc(cmd[0])
	if !found {
		return EmptyToken, ErrCommandNotFound(cmd[0].String)
	}

	// run proc
	ret, err := proc(interp, cmd)

	// decide if we exploded or not
	if err != nil && !errors.Is(err, ErrFlowControl) {
		err = fmt.Errorf("%s: %w", cmd[0].String, err)
	}

	if interp.calldepth == 1 && errors.Is(err, ErrReturn) {
		// don't pass an ErrReturn err all the way up the callstack
		err = nil
	}

	return ret, err
}

func (interp *Interp) Exec(cmd Command) (tok *Token, err error) {
	if interp.calldepth == 0 {
		interp.Mutex.Lock()
		defer interp.Mutex.Unlock()
	}

	interp.calldepth++
	defer func() {
		interp.calldepth--
		if x := recover(); x != nil {
			tok, err = EmptyToken, ErrGoPanic(x)
		}
	}()

	// try to head-off any stack-exploding
	if interp.calldepth >= interp.MaxCallDepth {
		return EmptyToken, ErrMaxCallDepthExceeded
	}

	// substitution pass
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

	return interp.ExecLiteral(args)
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

func (interp *Interp) ExecBytes(rawScript []byte) (*Token, error) {
	script, err := LexBytes(rawScript)
	if err != nil {
		return EmptyToken, err
	}
	return interp.ExecScript(script)
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
