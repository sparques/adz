package adz

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type ArgSet struct {
	Cmd, Help string
	ArgGroups []*ArgGroup
	Lazy      bool
	// PosOnly   bool
}

// NewArgSet returns an ArgSet with Cmd initialzed with name.
// If args are supplied, they are added to an ArgGroup and the
// the ArgGroup is added to ArgSet.
func NewArgSet(name string, args ...*Argument) *ArgSet {
	as := &ArgSet{
		Cmd:  name,
		Lazy: true,
	}
	if len(args) > 0 {
		as.ArgGroups = []*ArgGroup{
			NewArgGroup(args...),
		}
	}
	return as
}

// ArgGroup appends ag to ArgSet's ArgGroups
func (as *ArgSet) ArgGroup(ags ...*ArgGroup) {
	as.ArgGroups = append(as.ArgGroups, ags...)
}

// Return a slice of accepted arities.
// Not needed any more?
func (as *ArgSet) Arities() (arr []Arity) {
	for _, ag := range as.ArgGroups {
		arr = append(arr, ag.Arity())
	}
	return
}

// aritySummary nicely displays valid arities for help/error messages
func (as *ArgSet) aritySummary() string {
	var fixed []int
	for _, g := range as.ArgGroups {
		if !g.PosVariadic {
			fixed = append(fixed, len(g.Pos))
		}
	}
	sort.Ints(fixed)
	parts := make([]string, len(fixed))
	for i, n := range fixed {
		parts[i] = fmt.Sprint(n)
	}
	// single-group path never reaches here
	return strings.Join(parts, " | ")
}

// BindArgs uses the defined ArgSet to bind arguments passed in args to a map[string]*Token.
// This map[string]*Token is suitable for passing to interp.Push() as done when invoking
// a Proc.
func (as *ArgSet) BindArgs(interp *Interp, args []*Token) (boundArgs map[string]*Token, err error) {
	boundArgs = make(map[string]*Token)

	// Validate Ourself
	err = as.Validate()
	if err != nil {
		return
	}

	// No args given, no args required
	if len(args) < 2 && len(as.ArgGroups) == 0 {
		return
	}

	// put every argument in namedArgs or posArgs
	namedArgs, posArgs, err := ParseArgs(args)
	if err != nil {
		return
	}

	// figure out which ArgGroup to use
	arity := Arity(len(posArgs))
	ag := as.GetArgGroup(arity)
	if ag == nil {
		err = fmt.Errorf("expected arity to be one of %v, got %d", as.aritySummary(), arity)
		return
	}

	// if lazy matching is enabled, go through all the provided named args and
	// complete them or throw error
	// if lazy is enabled AND NamedVariadic is enabled, map to a name otherwise
	// ... just don't throw the error? Neat feature or footgun?
	if as.Lazy {
		for name, val := range namedArgs {
			var fullName string
			fullName, err = ag.lazyMatch(name)
			if err != nil {
				if ag.NamedVariadic {
					continue
				}
				return
			}
			if fullName != name {
				delete(namedArgs, name)
				namedArgs[fullName] = val
			}
		}
	}

	// cycle through named arguments expected,
	// arg.Get() will use default values, coerce procs
	// as necessary.
	for name, arg := range ag.Named {
		var val *Token
		val, err = arg.Get(interp, namedArgs[name])
		if err != nil {
			return
		}
		boundArgs[arg.Name[1:]] = val
	}

	// cycle through named arguments provided
	for name, val := range namedArgs {
		_, ok := ag.Named[name]
		if ok {
			continue
		}

		if !ag.NamedVariadic {
			err = ErrArgExtra(name)
			return
		}
		// not a specified arg, but NamedVariadic is true, so just bind it
		boundArgs[name[1:]] = val
	}

	// bind positional args
	for i := range ag.Pos {
		if i < len(posArgs) {
			boundArgs[ag.Pos[i].Name], err = ag.Pos[i].Get(interp, posArgs[i])
			if err != nil {
				return
			}
			continue
		}
		// got to here; there are more defined args than what was supplied
		// call Get with nil to attempt to get the Default
		boundArgs[ag.Pos[i].Name], err = ag.Pos[i].Get(interp, nil)
		if err != nil {
			return
		}
	}

	// do we have more more supplied args than defined?
	if len(posArgs) > len(ag.Pos) {
		// if variadic, just shove 'em all into args
		if ag.PosVariadic {
			boundArgs["args"] = NewList(posArgs[len(ag.Pos)-1:])
			// fin
			return
		}
		// not variadic, throw error
		// but really, we shouldn't be able to get here due to earlier processing
		err = ErrArgExtra(posArgs[len(ag.Pos)].String)
	}

	return
}

func (as *ArgSet) GetArgGroup(arr Arity) *ArgGroup {
	switch len(as.ArgGroups) {
	case 0:
		return &ArgGroup{}
	case 1:
		return as.ArgGroups[0]
	default:
		for _, ag := range as.ArgGroups {
			if !ag.PosVariadic && ag.Arity() == arr {
				return ag
			}
		}
	}
	// nil or &ArgGroup{}?
	return nil
}

// HelpText generates the entire help message
func (as *ArgSet) HelpText() string {
	msg := &strings.Builder{}
	msg.WriteString(as.Signature())
	if as.Help != "" {
		fmt.Fprintf(msg, "\n\n%s\n\n", as.Help)
	} else {
		fmt.Fprintf(msg, "\n\n")
	}
	// question: Should positional args be uniq'd together or should we just
	// show every help message?
	miniUsage := len(as.ArgGroups) > 1

	for _, ag := range as.ArgGroups {
		if miniUsage {
			fmt.Fprintf(msg, "\n%s %s\n", as.Cmd, ag.Prototype())
		}
		// Use Names() instead of the ag.Named map directly to get
		// a stable, sorted list
		for _, name := range ag.Names() {
			fmt.Fprintf(msg, "\t%s\n", ag.Named[name].HelpLine())
		}
		for _, pos := range ag.Pos {
			fmt.Fprintf(msg, "\t%s\n", pos.HelpLine())
		}
	}

	return msg.String()
}

// ParseProto parses a proc prototype (an *adz.Token) into an ArgSet.
// PasreProto is for generating an ArgSet from a prototype passed
// from adz.
// Procs written in go should just build out ArgSet within go.
func (as *ArgSet) ParseProto(proto *Token) error {
	// get rid of the old Arguments, just in case
	as.ArgGroups = []*ArgGroup{}

	protoList, err := proto.AsList()
	if err != nil {
		return err
	}

	// split protoList into a List of Lists based on |
	protoLists := ListOfLists(protoList, "|")

	for j := range protoLists {
		ag := NewArgGroup()

		for i, p := range protoLists[j] {
			switch {
			// single character parameters are valid, but can
			// never be a named argument, so nip this buffer
			// overflow in the bud
			case len(p.Index(0).String) < 2:
				arg, err := ParseProtoArg(p)
				if err != nil {
					return fmt.Errorf("arg %d: %w", i, err)
				}
				ag.Pos = append(ag.Pos, arg)
			case p.Index(0).String[0] == '-':
				arg, err := ParseProtoArg(p)
				if err != nil {
					return fmt.Errorf("arg %d: %w", i, err)
				}
				if arg.Name == "-args" {
					ag.NamedVariadic = true
				} else {
					ag.Named[arg.Name] = arg
				}
			default:
				arg, err := ParseProtoArg(p)
				if err != nil {
					return fmt.Errorf("arg %d: %w", i, err)
				}
				ag.Pos = append(ag.Pos, arg)
				if arg.Name == "args" {
					ag.PosVariadic = true
				}
			}
		}

		as.ArgGroups = append(as.ArgGroups, ag)
	}

	return nil
}

// ShowUsage is a convenience for writing the full HelpText to w.
func (as *ArgSet) ShowUsage(w io.Writer) {
	w.Write([]byte(as.HelpText()))
}

// Signature generates the command along with the arg prototype
func (as *ArgSet) Signature() string {
	usage := &strings.Builder{}

	// show command
	usage.WriteString(as.Cmd)
	// fmt.Sprintf(usage, "%s", as.Cmd)

	// sort by arity for consistent output
	groups := append([]*ArgGroup(nil), as.ArgGroups...)
	sort.Slice(groups, func(i, j int) bool {
		ai, aj := groups[i].Arity(), groups[j].Arity()
		if ai == aj {
			return i < j
		} // keep original order to break ties
		return ai < aj || aj == -1 // put fixed before variadic
	})

	separator := false
	for _, ag := range groups {
		if separator {
			fmt.Fprintf(usage, "  |")
		}

		// show named args first using Names() to
		// get a sorted list.
		for _, name := range ag.Names() {
			fmt.Fprintf(usage, "  %s", quoted(ag.Named[name].String()))
		}

		// then positional
		for _, pos := range ag.Pos {
			fmt.Fprintf(usage, "  %s", quoted(pos.String()))
		}
		separator = true
	}

	return usage.String()
}

// Validate makes sure the ArgSet is sane:
// that Arguments arities are all correct,
// that it is either PosVariadic or multi-arity or neither.
func (as *ArgSet) Validate() error {
	if len(as.ArgGroups) == 0 {
		return fmt.Errorf("%s: no arg groups defined", as.Cmd)
	}
	// flip variadic flags if needed
	for _, ag := range as.ArgGroups {
		if len(ag.Pos) > 0 && ag.Pos[len(ag.Pos)-1].Name == "args" {
			ag.PosVariadic = true
		}
		if _, ok := ag.Named["-args"]; ok {
			ag.NamedVariadic = true
			// actually having an Argument named "-args" is not deseriable
			// setting the flag is good enough.
			delete(ag.Named, "-args")

		}
		// Coerce-without-default rule ({} means “must supply a value” if you use it)
		for _, a := range ag.Named {
			if a.Coerce != nil && a.Default == nil {
				return fmt.Errorf("%s: %s has coercer but no default; use {} to require a value", as.Cmd, a.Name)
			}
		}
	}

	if len(as.ArgGroups) == 1 {
		// single-group mode: allow defaults and variadic
		// give "args" a default of empty list
		posCnt := len(as.ArgGroups[0].Pos)
		if posCnt > 0 && as.ArgGroups[0].Pos[posCnt-1].Name == "args" {
			as.ArgGroups[0].Pos[posCnt-1].Default = EmptyToken
		}
		return nil
	}

	// multi-arity mode: no variadic, no positional defaults, unique fixed arities
	seen := map[Arity]struct{}{}
	for i, ag := range as.ArgGroups {
		if ag.PosVariadic {
			return fmt.Errorf("%s: cannot use variadic positional with multi-arity (group %d)", as.Cmd, i)
		}
		for _, p := range ag.Pos {
			if p.Default != nil {
				return fmt.Errorf("%s: positional defaults not allowed with multi-arity (group %d)", as.Cmd, i)
			}
		}
		ar := Arity(len(ag.Pos))
		if _, dup := seen[ar]; dup {
			return fmt.Errorf("%s: duplicate fixed arity %d in multi-arity", as.Cmd, ar)
		}
		seen[ar] = struct{}{}
	}
	return nil
}

type ArgGroup struct {
	Named                      map[string]*Argument
	Pos                        []*Argument
	PosVariadic, NamedVariadic bool
}

func NewArgGroup(args ...*Argument) *ArgGroup {
	ag := &ArgGroup{
		Named: make(map[string]*Argument),
		Pos:   []*Argument{},
	}
	for i := range args {
		if strings.HasPrefix(args[i].Name, "-") {
			ag.Named[args[i].Name] = args[i]
		} else {
			ag.Pos = append(ag.Pos, args[i])
		}
	}
	return ag
}

func (ag *ArgGroup) Arity() Arity {
	// if no Pos args are accepted, this is Arity of 0
	if len(ag.Pos) == 0 {
		return 0
	}
	// if ag.Pos has an 'args' argument return -1 to indicate
	// this is a multi-Arity ArgGroup
	if ag.PosVariadic {
		return -1
	}

	return Arity(len(ag.Pos))
}

func (ag *ArgGroup) Names() (acc []string) {
	acc = []string{}
	for name := range ag.Named {
		acc = append(acc, name)
	}
	sort.Strings(acc)
	return acc
}

func (ag *ArgGroup) GetNamed(name string) *Argument {
	return ag.Named[name]
}

func (ag *ArgGroup) lazyMatch(name string) (fullName string, err error) {
	var found bool
	names := ag.Names()
	// first try looking for an exact match and return if found
	for i := range names {
		if names[i] == name {
			return name, nil
		}
	}
	// otherwise, try a lazy match
	for i := range names {
		if strings.HasPrefix(names[i], name) {
			if found {
				// already found? tsk tsk
				return "", fmt.Errorf("%s is ambiguous: %s/%s", name, names[i], fullName)
			}
			fullName = names[i]
			found = true
		}
	}

	if !found {
		if ag.NamedVariadic {
			return name, nil
		}
		return "", ErrArgExtra(name)
	}

	return
}

// Prototype shows the prototype for the ArgGroup
func (ag *ArgGroup) Prototype() string {
	usage := &strings.Builder{}

	// show named args first
	for _, named := range ag.Named {
		fmt.Fprintf(usage, "  %s", quoted(named.String()))
	}

	// then positional
	for _, pos := range ag.Pos {
		fmt.Fprintf(usage, "  %s", quoted(pos.String()))
	}

	return usage.String()
}

type Argument struct {
	Name    string
	Default *Token
	Coerce  *Token
	Help    string
}

type Arity int

func (arg *Argument) Get(interp *Interp, tok *Token) (ret *Token, err error) {
	ret = tok

	if ret == nil {
		ret = arg.Default
	}

	// if we're nil here, that means we weren't given an argument and there's no default
	if ret == nil {
		err = ErrArgMissing(arg.Name)
		return
	}

	if arg.Coerce == nil || arg.Coerce.String == "" {
		return ret, nil
	}

	coerceCmd, _ := arg.Coerce.AsList()
	coerceCmd = append(coerceCmd, ret)
	ret, err = interp.Exec(coerceCmd)
	if err != nil {
		err = fmt.Errorf("arg %s: %w", arg.Name, err)
	}
	return
}

func (arg *Argument) String() string {
	b := &strings.Builder{}
	b.WriteString(arg.Name)

	if arg.Default == nil && arg.Coerce != nil {
		b.WriteString(` {}`)
	}

	if arg.Default != nil {
		b.WriteString(" " + arg.Default.Quoted())
	}

	if arg.Coerce == nil {
		return b.String()
	}

	b.WriteString(` `)
	b.WriteString(arg.Coerce.Quoted())

	return b.String()
}

// HelpLine returns the argument name, and if it exists, its help text, default value and coerce.
func (arg *Argument) HelpLine() string {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "%s\t%s", arg.Name, arg.Help)
	if arg.Coerce != nil && arg.Coerce.String != "" {
		fmt.Fprintf(builder, " (%s)", arg.Coerce.String)
	}
	if arg.Default != nil {
		if arg.Default.String == "" && arg.Coerce != nil {
			fmt.Fprintf(builder, " (REQUIRED)")
		} else {
			fmt.Fprintf(builder, " (Default: %s)", quoted(arg.Default.String))
		}
	} else {
		fmt.Fprintf(builder, " (REQUIRED)")
	}

	return builder.String()
}

// ParseArgs takes a slice of adz *Tokens and parses them per this rule set:
//
//   - The first argument is skipped; this is assumed to be the command.
//   - An argument starting with a dash is a named argument.
//   - The token after a named arguement is the value.
//   - It is an error to have a named argument without a following value argument.
//   - If the argument does not start with a dash, it is a positional argument.
//   - After an argument of -- is passed, all subsequent arguments including those
//     that start with a dash will be treated as positional arguments.
//   - Single or zero character arguments are always positional.
func ParseArgs(args []*Token) (namedArgs map[string]*Token, posArgs []*Token, err error) {
	posArgs = []*Token{}
	namedArgs = map[string]*Token{}

	// iterate over args,
	for i := 1; i < len(args); i++ {
		// flag to stop interpretting leading dash as named argument
		if args[i].String == "--" {
			// stop processing; append remaining variables as positional args
			if i+1 < len(args) {
				posArgs = append(posArgs, args[i+1:]...)
			}
			break
		}
		if !strings.HasPrefix(args[i].String, "-") || len(args[i].String) < 2 {
			// positional arg
			posArgs = append(posArgs, args[i])
			continue
		}
		if i+1 >= len(args) {
			err = fmt.Errorf("argument %s: %w", args[i].String, ErrExpectedMore)
			return
		}

		namedArgs[args[i].String] = args[i+1]
		i++ // skip value we just assigned
	}

	return
}

// not sure if this is useful anywhere else... just leave it here for now
func ListOfLists(list List, separator string) (lol []List) {
	var i, j int
	for j < len(list) {
		if list[j].String == separator {
			lol = append(lol, list[i:j])
			i = j + 1
			j++
			continue
		}
		j++
	}
	lol = append(lol, list[i:j])
	return
}

func ParseProtoArg(arg *Token) (parg *Argument, err error) {
	list, err := arg.AsList()
	if err != nil {
		return nil, err
	}
	parg = &Argument{}
	switch len(list) {
	case 4:
		parg.Help = list[3].String
		fallthrough
	case 3:
		parg.Coerce = list[2]
		if len(list) > 3 && parg.Coerce.String == "" {
			parg.Coerce = nil
		}
		fallthrough
	case 2:
		// if you have a coerce proc specified,
		// and a default of empty string, it will not be
		// treated as though it has a default. If you have only a
		// name and default/ specified, then an empty string is the default.
		// This is admittedly a little confusing and counter intuitive,
		// but it nets the functionality desired and hopefully no notices
		// it's weird / not great design.
		parg.Default = list[1]
		if len(list) > 2 && parg.Default.String == "" {
			parg.Default = nil
		}
		fallthrough
	case 1:
		parg.Name = list[0].String
		return
	case 0:
		return nil, fmt.Errorf("empty arg?")
	}

	return nil, fmt.Errorf("too many elements in arg proto")
}

func Flags(arg ...*Argument) map[string]*Argument {
	named := make(map[string]*Argument)
	for _, a := range arg {
		named[a.Name] = a
	}
	return named
}

func Arg(name string) *Argument {
	return &Argument{Name: name}
}

func ArgHelp(name, help string) *Argument {
	return &Argument{Name: name, Help: help}
}

func ArgDefault(name string, def *Token) *Argument {
	return &Argument{Name: name, Default: def}
}
