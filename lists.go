package adz

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

var ListLib = map[string]Proc{}

func init() {
	// keep list in StdLib
	StdLib["list"] = ProcList

	// consider making list::new return a list *object*
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
	ListLib["contains"] = ProcListContains
	ListLib["split"] = ProcListSplit
	ListLib["find"] = ProcListFind
}

func (l List) Proc(interp *Interp, args []*Token) (*Token, error) {
	// args[0] should be the list as well
	if len(args) <= 1 {
		return args[0], nil
	}
	switch args[1].String {
	case "append":
		newList := append(slices.Clone(l), args[2:]...)
		return NewList(newList), nil
	case "len":
		return NewToken(len(l)), nil
	case "reverse":
		l = slices.Clone(l)
		slices.Reverse(l)
		return NewList(l), nil
	default:
		// index operation?
		if i, err := args[1].AsInt(); err == nil {
			return args[0].Index(i), nil
		}
	}

	return args[0], nil
}

// func ProcListNew(interp *Interp, args []*Token) (*Token, error) {
//
// }

// ProcList returns a well-formed list. The list is pre-parsed
// as a list and will be readily accessible for use as a list.
func ProcList(interp *Interp, args []*Token) (*Token, error) {
	switch len(args) {
	case 1:
		return NewList([]*Token{}), nil
	case 2:
		return NewList([]*Token{args[1]}), nil
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
	as := NewArgSet("list::map",
		&Argument{
			Name:    "-skiperrors",
			Help:    "If true, any errors encountered while calling {proc} are simply skipped (as though continue were called) rather than stopping with an error.",
			Default: FalseToken,
			Coerce:  Proc(ProcBool).AsToken("bool"),
		},
		ArgHelp("list", "The list to iterate over."),
		ArgHelp("proc", "The proc to call for each list element."),
	)
	as.Help = "Iterates over {list} calling {proc} with each element from {list} appended to it. A new list is generated from the return values from calling {proc}. The returned list has the same number of elements as {list}. If {proc} returns by calling [continue], that element is skipped; i.e. the returned list is one element shorter than {list} for each time that [continue] is used. If {proc} returns using [break], the list is truncated at that element."

	parsedArgs, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}

	list, err := parsedArgs["list"].AsList()
	if err != nil {
		return EmptyToken, err
	}

	outList := make([]*Token, 0, len(list))

	cmdPrefix, _ := parsedArgs["proc"].AsCommand()

	// look up proc and call it directly to avoid interp.Exec()'s
	// substitution pass.
	proc, found := interp.getProc(cmdPrefix[0])
	if !found {
		return EmptyToken, ErrCommandNotFound(cmdPrefix[0].String)
	}

	for _, e := range list {
		cmd := append(cmdPrefix, e) // need to quote
		// c
		// ret, err := interp.Exec(cmd)
		ret, err := proc(interp, cmd)
		switch err {
		case nil:
			outList = append(outList, ret)
		case ErrContinue:
			// skip this element
		case ErrBreak:
			// truncate result and return without error
			return NewList(outList), nil
		default:
			if parsedArgs["skiperrors"].Data.(bool) {
				continue
			}
			return EmptyToken, err
		}
	}

	return NewList(outList), nil
}

func ProcListUniq(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		&Argument{
			Name: "list",
			Help: "A list.",
		})
	as.Help = "Returns a list that has replaced consecutive runs of equal elements with a single copy."
	parsedArgs, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	// shallow uniq? just .String and not .Data? means true and on won't match...
	list, err := parsedArgs["list"].AsList()
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

func ProcListContains(interp *Interp, args []*Token) (*Token, error) {
	list, err := args[1].AsList()
	if err != nil {
		return EmptyToken, err
	}
	for e := range list {
		if list[e].Equal(args[2]) {
			return TrueToken, nil
		}
	}

	return FalseToken, nil
}

// TODO: split on multiple substrs.
// TODO: Check args
func ProcListSplit(interp *Interp, args []*Token) (*Token, error) {
	// as:=NewArgSet(args[0].String,
	// 	ArgHelp("list","list to be split"),
	// 	ArgHelp("split","string used as delimiter")
	list := splitOnSubstr(args[1].String, args[2].String)
	return NewToken(NewTokenListString(list)), nil
}

func splitOnSubstr(str, substr string) []string {
	var (
		before, after string
		found         bool = true
		acc                = []string{}
	)
	for found {
		before, after, found = strings.Cut(str, substr)
		acc = append(acc, before)
		str = after
	}
	return acc
}

func ProcListFind(interp *Interp, args []*Token) (*Token, error) {
	// list::find -type {match type} -matchcase {bool, false} list pattern
	as := NewArgSet(args[0].String,
		&Argument{
			Name:    "-type",
			Help:    "Specifies how to use {pattern} to match elements of {list}. ",
			Default: NewToken("glob"),
			Coerce:  NewToken("tuple {exact glob regex}"),
		},
		&Argument{
			Name:    "-matchcase",
			Help:    "Whether or not to be case sensitive in matching.",
			Default: TrueToken,
			Coerce:  Proc(ProcBool).AsToken("bool"),
		},
		&Argument{
			Name: "list",
			Help: "The list within which to find elements.",
		},
		&Argument{
			Name: "pattern",
			Help: "What to search {list} for.",
		},
	)
	as.Help = "Iterate over {list}, returning a new list whose elements match elements of {list} as dictated by {pattern}."

	parsedArgs, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}

	list, _ := parsedArgs["list"].AsList()
	out := make([]*Token, 0, len(list))

	pattern := parsedArgs["pattern"].String
	switch parsedArgs["type"].String {
	case "exact":
		if parsedArgs["matchcase"].Data.(bool) {
			for i := range list {
				if list[i].Equal(parsedArgs["pattern"]) {
					out = append(out, list[i])
				}
			}
		} else {
			for i := range list {
				if strings.ToLower(list[i].String) == strings.ToLower(pattern) {
					out = append(out, list[i])
				}
			}
		}
	case "glob":
		if parsedArgs["matchcase"].Data.(bool) {
			for i := range list {
				if match, _ := filepath.Match(
					parsedArgs["pattern"].String,
					list[i].String); match {
					out = append(out, list[i])
				}
			}
		} else {
			pattern = strings.ToLower(pattern)
			for i := range list {
				if match, _ := filepath.Match(
					pattern,
					strings.ToLower(list[i].String)); match {
					out = append(out, list[i])
				}
			}
		}
	case "regex":
		return EmptyToken, ErrNotImplemented
	}

	return NewList(out), nil
}

/*
func ProcListJoin(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(
		&Argument{
			Name: "list",
		},
		&Argument{
			Name: "joinString",
		},
	)

	// parsedArgs, err := as.Parse(args[1:])
}
*/
