package adz

import (
	"fmt"
	"path/filepath"
	"strings"
)

func init() {
	StdLib["set"] = ProcSet
	StdLib["del"] = ProcDel
	StdLib["subst"] = ProcSubst
	StdLib["var"] = ProcVar
	StdLib["import"] = ProcImport
}

func ProcSet(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	return interp.SetVar(args[1].String, args[2])
}

func ProcDel(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum(1, len(args)-1)
	}

	return interp.DelVar(args[1].String)
}

func ProcSubst(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}
	return interp.Subst(args[1])
}

//  ProcIdx (equiv to lindex

// var varname - returns true/false if varname exists or doesn't exist
// var varname {val} sets varname to val
// var varname cmd <args> does a variable subcommand like...
// var varname idx n ;# treat varname as a list and return the nth index of varname;
// var varname idx {n1 n2 n3...} ;# treat varname as a list and return the n3-th index of the n2-th index of the n1-th index of varname;
// var varname idx n val;# treat varname as a list and set the nth index of varname to val
// var varname len ;# treat varname as a list and return its length
// var varname append ;# append
func ProcVar(interp *Interp, args []*Token) (*Token, error) {
	switch len(args) {
	case 1:
		// var command by itself, list out vars
		out := make([]*Token, 0, len(interp.Frame.localVars))
		for k, v := range interp.Frame.localVars {
			out = append(out, NewList([]*Token{NewTokenString(k), v}))
		}
		return NewList(out), nil
	case 2: // var <varname> ; return true/false if var exists
		if _, ok := interp.Frame.localNamespace.Vars[args[1].String]; ok {
			return TrueToken, nil
		}
		return FalseToken, nil
	default:
		vari, err := interp.GetVar(args[1].String)
		if err != nil {
			return EmptyToken, err
		}
		switch args[2].String {
		case "len":
			variList, err := vari.AsList()
			if err != nil {
				return NewTokenInt(1), nil
			}
			return NewTokenInt(len(variList)), nil
		case "idx":
			if len(args) != 4 {
				return EmptyToken, ErrArgCount(3, len(args)-1)
			}
			idxList, err := args[3].AsList()
			if err != nil {
				return EmptyToken, ErrSyntax
			}

			for _, idxTok := range idxList {
				idx, err := idxTok.AsInt()
				if err != nil {
					return EmptyToken, ErrExpectedInt
				}
				vari = vari.Index(idx)
				// if vari == EmptyToken {
				// 	break
				// }
			}

			return vari, nil

		}
	}

	return EmptyToken, nil

}

func importProc(interp *Interp, procName string) error {
	if !strings.HasPrefix(procName, "::") {
		return fmt.Errorf("import name must be fully qualified")
	}
	ns, id, err := interp.ResolveIdentifier(procName, false)
	if err != nil {
		return err
	}
	proc, ok := ns.Procs[id]
	if !ok {
		return fmt.Errorf("proc %s does not exist", ns.Qualified(id))
	}

	interp.Frame.localNamespace.Procs[id] = proc

	return nil
}

// ProcImport implements the import proc.
// With one arg, import climbs the stack, looking for a variable
// with the same name and puts it into the localvars.
// With two vars, the first must be a fully qualified name. This
// var is linked as the
//
// Globs only for fully qualified identifiers.
// import -proc ::list::* ;# import all procs in ::list namespace
// import ::ns::* ;# import all variables from ::ns name space
//
//	combined:
//
// import -proc {::list::idx ::list::someVar} -proc ::list::len
// import -proc {::list::idx ::list::len}
func ProcImport(interp *Interp, args []*Token) (*Token, error) {
	// parsedArgs, err := ParseArgsWithProto(`{-proc {}} {-var {}}`, args[1:])
	parsedArgs, err := ParseArgsWithProto(`{-proc {}} {-var {}} {-file {}}`, args[1:])
	if err != nil {
		return EmptyToken, err
	}
	// import files first so var and proc namespace importing works
	// but do not hardcode calls to os package.

	// treat the values of proc and var as lists, iterate over them
	procList, err := parsedArgs["proc"].AsList()
	if err != nil {
		return EmptyToken, fmt.Errorf("could not parse -proc argument as list: %w", err)
	}
	for _, procName := range procList {
		ns, id, err := interp.ResolveIdentifier(procName.String, false)
		if err != nil {
			return EmptyToken, fmt.Errorf("could not resolve %s: %w", procName.String, err)
		}
		for p := range ns.Procs {
			if m, _ := filepath.Match(id, p); m {
				err = importProc(interp, ns.Qualified(p))
				if err != nil {
					return EmptyToken, fmt.Errorf("could not import %s: %w", ns.Qualified(id), err)
				}
			}
		}
	}

	// import vars
	varList, err := parsedArgs["var"].AsList()
	if err != nil {
		return EmptyToken, err
	}
	for _, varPair := range varList {
		ref := &Ref{}
		varName := varPair.Index(0).String
		as := varPair.Index(1).String

		ref, err := interp.getVarRef(varName)
		if err != nil {
			return EmptyToken, err
		}
		if as == "" {
			_, as = identifierParts(varName)
		}
		interp.Frame.localVars[as] = ref.Token()
	}
	return EmptyToken, nil
}

func (interp *Interp) getVarRef(varName string) (ref *Ref, err error) {
	ref = &Ref{}

	// for fully qualified var names, we want to error out if the namesapce doesn't
	// exist, but want to create the variable if the namespace exists and the
	// variable does not yet.
	if strings.HasPrefix(varName, "::") {
		ns, id, err := interp.ResolveIdentifier(varName, false)
		if err != nil {
			return nil, fmt.Errorf("could not resolve %s: %w", varName, err)
		}
		_, ok := ns.Vars[id]
		if !ok {
			// doesn't exist yet, create it
			ns.Vars[id] = NewToken("")
		}
		ref.Name = id
		ref.Namespace = ns
		return ref, nil
	}

	// non-fully qualified variable given; check local Namespace first
	// and then climb the stack to find it
	_, ok := interp.Frame.localNamespace.Vars[varName]
	if ok {
		ref.Name = interp.Frame.localNamespace.Qualified(varName)
		ref.Namespace = interp.Frame.localNamespace
		return ref, nil
	}

	// ascend stack until we find the variable
	for i := len(interp.Stack) - 1; i >= 0; i-- {
		// try to match a ns var first
		_, ok := interp.Stack[i].localNamespace.Vars[varName]
		if ok {
			ref.Name = varName
			ref.Namespace = interp.Stack[i].localNamespace
			return ref, nil
		}
		_, ok = interp.Stack[i].localVars[varName]
		if ok {
			ref.Name = varName
			ref.Frame = interp.Stack[i]
			return ref, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNoVar, varName)
}
