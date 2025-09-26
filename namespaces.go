package adz

import (
	"fmt"
	"strings"
)

func init() {
	StdLib["namespace"] = ProcNamespace
}

type Namespace struct {
	Name  string
	Vars  map[string]*Token
	Procs map[string]Proc
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:  name,
		Vars:  make(map[string]*Token),
		Procs: make(map[string]Proc),
	}
}

// Namespace
func ProcNamespace(interp *Interp, args []*Token) (*Token, error) {
	switch len(args) {
	case 1:
		return NewTokenString(interp.Frame.localNamespace.Qualified("")), nil
	case 3:
	default:
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	ns, _, err := interp.ResolveIdentifier(args[1].String+"::", true)
	if err != nil {
		return EmptyToken, fmt.Errorf("%s: %w", args[1].String, err)
	}

	interp.Push(&Frame{
		localNamespace: ns,
		localVars:      ns.Vars,
		localProcs:     ns.Procs,
		namespaceRoot:  true,
	})
	defer interp.Pop()
	return interp.ExecToken(args[2])
}

func identifierParts(id string) (namespace, name string) {
	i := strings.LastIndex(id, "::")
	if i == -1 {
		return "", id
	}
	return id[:i], id[i+2:]
}

func (interp *Interp) ResolveIdentifier(id string, create bool) (*Namespace, string, error) {
	// first strip a possible leading $
	id = strings.TrimPrefix(id, "$")
	if !isQualified(id) {
		// no namespace separators, use current namespace
		return interp.Frame.localNamespace, id, nil
	}
	ns, name := identifierParts(id)
	ns = strings.TrimPrefix(ns, "::")

	namespace, ok := interp.Namespaces[ns]
	if !ok {
		if create {
			interp.Namespaces[ns] = NewNamespace(ns)
			return interp.Namespaces[ns], name, nil
		}
		return nil, id, ErrNoNamespace(ns)
	}

	return namespace, name, nil
}

// Qualified takes id and returns a fully qualified identifier
func (ns *Namespace) Qualified(id string) string {
	// TODO: handle colons in id
	if isQualified(id) {
		return id
	}

	switch {
	case id == "":
		// asking for the fully-qualified namespace name
		return "::" + ns.Name
	case ns.Name == "":
		// special exception for global namespace
		return "::" + id
	default:
		return "::" + ns.Name + "::" + id
	}
}

func isQualified(id string) bool {
	return strings.LastIndex(id, "::") != -1
}
