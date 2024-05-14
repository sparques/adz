package adz

import "sort"

func init() {
	StdLib["list"] = ProcList
	StdLib["concat"] = ProcConcat
	StdLib["len"] = ProcLen
	StdLib["slice"] = ProcSlice
	StdLib["sort"] = ProcSort
	StdLib["idx"] = ProcIdx
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

// TODO: add various options like -type <numeric> and -order <reverse>
func ProcSort(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	list, _ := args[1].AsList()

	sort.Sort(List(list))

	return NewList(list), nil
}
