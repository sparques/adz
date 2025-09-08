package adz

import (
	"errors"
	"fmt"
	"slices"
	"sort"
)

var ListLib = map[string]Proc{}

func init() {
	// keep list in StdLib
	StdLib["list"] = ProcList

	ListLib["new"] = ProcList
	ListLib["concat"] = ProcConcat
	ListLib["len"] = ProcLen
	ListLib["slice"] = ProcSlice
	ListLib["sort"] = ProcSort
	ListLib["idx"] = ProcIdx
	ListLib["setidx"] = ProcListSet
	ListLib["map"] = ProcListMap
	ListLib["reverse"] = ProcListReverse
	ListLib["uniq"] = ProcListUniq
	ListLib["append"] = ProcListAppend
	ListLib["anoint"] = ProcListAnoint
}

// ProcList returns a well-formed list. The list is pre-parsed
// as a list and will be readily accessible for use as a list.
func ProcList(interp *Interp, args []*Token) (*Token, error) {
	switch len(args) {
	case 1:
		return EmptyToken, nil
	case 2:
		return args[1], nil
	default:
		return NewList(args[1:]), nil
	}
}

// ProcConcat returns its arguments concatenated
func ProcConcat(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, nil
	}
	var out string = stripLiteralBrackets(args[1].String)
	for _, tok := range args[2:] {
		out += " " + stripLiteralBrackets(tok.String)
	}
	return &Token{String: out}, nil
}

// ProcLSet -- list setting proc better name? better functionality?

// ProcLindex -- lidx?

// implement lists as object? when looking up procs, check for a var of the same name and provide standard set of operations?

// a generic var command?

// var <varname> index <num>
// var <varname> set <val>
// var <varname> set <list of numbers or strings> <val>

// can we intersperse dictionaries and lists?
// var <varname> set {1 name last} Hiles
// if varname is {{name {first Jane Last Doe}} {name {first John last Doe}}} then the top returns a new dict with the value... {{name {first Jane Last Doe}} {name {first John last Hiles}}}

// calling the varname directly would be an alias to the var command
// easily implemetable by a check for varname existence and then
// calling ProcVar

// the alias is nice and I do like hiding away all those variable operations under a single command

func ProcLen(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	l, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}

	return NewTokenInt(len(l)), nil
}

func ProcSlice(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 4 {
		return EmptyToken, ErrArgCount(3, len(args)-1)
	}

	start, err := args[2].AsInt()
	if err != nil {
		return EmptyToken, ErrExpectedInt
	}
	end, err := args[3].AsInt()
	if err != nil {
		return EmptyToken, ErrExpectedInt
	}

	return args[1].Slice(start, end), nil
}

func ProcIdx(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}
	vari := args[1]
	idxList, err := args[2].AsList()
	if err != nil {
		return EmptyToken, ErrSyntax
	}

	for _, idxTok := range idxList {
		idx, err := idxTok.AsInt()
		if err != nil {
			return EmptyToken, ErrExpectedInt
		}
		vari = vari.Index(idx)
	}

	return vari, nil
}

func ProcListSet(interp *Interp, args []*Token) (*Token, error) {
	// list::set <list> {indices} value
	type step struct {
		parent *Token
		i      int
	}
	var chain []step

	listName := args[1].String
	list, err := interp.GetVar(listName)
	if err != nil {
		return EmptyToken, err
	}

	indices, err := args[2].AsList()
	if err != nil {
		return EmptyToken, fmt.Errorf("arg 2: %w", err)
	}

	for _, idxTok := range indices {
		i, err := idxTok.AsInt()
		if err != nil {
			return EmptyToken, fmt.Errorf("arg 2: %w", err)
		}
		chain = append(chain, step{parent: list, i: i})
		list = list.Index(i)
	}

	slices.Reverse(chain)

	// carry the updated node upward through each parent
	acc := args[3]
	for _, st := range chain {
		var e error
		acc, e = st.parent.IndexSet(st.i, acc)
		if e != nil {
			return EmptyToken, e
		}
	}
	return interp.SetVar(listName, acc)
}

// TODO: add various options like -type <numeric> and -order <reverse>
func ProcSort(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	list, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}

	sort.Sort(List(list))

	return NewList(list), nil
}

func ProcListReverse(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	list, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}
	list = slices.Clone(list)
	slices.Reverse(list)
	newList := NewList(list)
	return newList, nil
}

func ProcListMap(interp *Interp, args []*Token) (*Token, error) {
	// run proc against each element, taking the output to make a new list
	// any error aborts
	// break stops processing but doesn't throw an error
	// list::map <list> <proc>
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	list, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}

	outList := make([]*Token, 0, len(list))

	for _, e := range list {
		ret, err := interp.Exec([]*Token{args[2], e})
		switch err {
		case nil:
			outList = append(outList, ret)
		case ErrContinue:
			// skip this element
		case ErrBreak:
			// truncate result and return without error
			return NewList(outList), nil
		default:
			return EmptyToken, err
		}
	}

	return NewList(outList), nil
}

func ProcListUniq(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	// shallow uniq? just .String and not .Data? means true and on won't match...
	list, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}

	compactList := slices.CompactFunc(list, func(a *Token, b *Token) bool {
		return a.String == b.String
	})

	return NewList(compactList), nil
}

func ProcListAppend(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2)
	}

	listVar, err := interp.GetVar(args[1].String)
	if err != nil {
		// bubble up any error that isn't ErrNoVar
		if !errors.Is(err, ErrNoVar) {
			return EmptyToken, err
		}
		// just create a new tok
		listVar = NewTokenString("")
	}
	list, err := listVar.AsList()
	if err != nil {
		return EmptyToken, err
	}

	newList := slices.Clone(list)
	newList = append(newList, args[2:]...)

	return interp.SetVar(args[1].String, NewList(newList))
}

// ProcListAnoint implements list::anoint. anoint makes a list variable
// into a callable proc wherein list operations can be performed
func ProcListAnoint(interp *Interp, args []*Token) (*Token, error) {
	for i := 1; i < len(args); i++ {
		varName := args[i].String
		listVar, err := interp.GetVar(varName)
		switch {
		case err == nil:
			// no error, keep going
		case errors.Is(err, ErrNoVar):
			listVar = NewToken("")
			interp.SetVar(args[i].String, listVar)
		default:
			return EmptyToken, err
		}
		listVar.Data = Proc(func(interp *Interp, args []*Token) (*Token, error) {
			switch args[1].String {
			case "idx":
				listVar, _ := interp.GetVar(varName)
				args[0], args[1] = NewTokenString(varName), listVar
				return ProcIdx(interp, args)
			default:
				return EmptyToken, ErrCommandNotFound
			}
		})
	}

	return NewList(args[1:]), nil
}
