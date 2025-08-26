package adz

import "strings"

func init() {
	StdLib["namespace"] = ProcNamespace
}

// Namespace
func ProcNamespace(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	ns, ok := interp.Namespaces[args[1].String]
	if !ok {
		interp.Namespaces[args[1].String] = NewNamespace(args[1].String)
		ns = interp.Namespaces[args[1].String]
	}

	prevNS := interp.Namespace
	prevVars := interp.Vars
	defer func() {
		interp.Namespace = prevNS
		interp.Vars = prevVars
	}()
	interp.Namespace = ns
	interp.Vars = ns.Vars
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
	if strings.LastIndex(id, "::") == -1 {
		// no namespace separators, use current namespace
		return interp.Namespace, id, nil
	}
	ns, name := identifierParts(id)

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
