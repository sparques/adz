package adz

func init() {
	StdLib["list"] = ProcList
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
