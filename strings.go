package adz

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// --- registration -----------------------------------------------------------

var StringsProcs = map[string]Proc{
	"format":       ProcStringsFormat,
	"contains":     ProcStringsContains,
	"containsany":  ProcStringsContainsAny,
	"containsrune": ProcStringsContainsRune,
	"equalfold":    ProcStringsEqualFold,
	"hasprefix":    ProcStringsHasPrefix,
	"hassuffix":    ProcStringsHasSuffix,
	"count":        ProcStringsCount,
	"index":        ProcStringsIndex,
	"lastindex":    ProcStringsLastIndex,
	"indexany":     ProcStringsIndexAny,
	"lastindexany": ProcStringsLastIndexAny,
	"indexrune":    ProcStringsIndexRune,
	"split":        ProcStringsSplit,
	"splitn":       ProcStringsSplitN,
	"splitafter":   ProcStringsSplitAfter,
	"splitaftern":  ProcStringsSplitAfterN,
	"join":         ProcStringsJoin,
	"replace":      ProcStringsReplace,
	"replaceall":   ProcStringsReplaceAll,
	"repeat":       ProcStringsRepeat,
	"tolower":      ProcStringsToLower,
	"toupper":      ProcStringsToUpper,
	"totitle":      ProcStringsToTitle,
	"trim":         ProcStringsTrim,
	"trimleft":     ProcStringsTrimLeft,
	"trimright":    ProcStringsTrimRight,
	"trimspace":    ProcStringsTrimSpace,
	"trimprefix":   ProcStringsTrimPrefix,
	"trimsuffix":   ProcStringsTrimSuffix,
	"compare":      ProcStringsCompare,
	"map":          ProcStringsMap,
	"fields":       ProcStringsFields,
	"fieldsfunc":   ProcStringsFieldsFunc,
}

// Optional: convenience loader
func LoadStringsProcs(interp *Interp) {
	interp.LoadProcs("str", StringsProcs)
}

/*
func init() {
	StdLib["match"] = procMatch
}

func procMatch(interp *Interp, args []*Token) (*Token, error) {
	namedProto, posProto, err := ParseProto(NewToken(`{-style glob} {-matchcase true} pattern str`))
	if err != nil {
		return EmptyToken, err
	}

	parsedArgs, err := ParseArgs(namedProto, posProto, args[1:])
	if err != nil {
		return EmptyToken, err
	}

	pattern, str := parsedArgs["pattern"].String, parsedArgs["str"].String
	if !parsedArgs["matchcase"].IsTrue() {
		pattern = strings.ToLower(pattern)
		str = strings.ToLower(str)
	}

	switch parsedArgs["style"].String {
	case "substr":
		return NewToken(strings.Contains(str, pattern)), nil
	case "regex":
		return EmptyToken, ErrNotImplemented("regex")
	case "glob":
		m, err := filepath.Match(pattern, str)
		if err != nil {
			return EmptyToken, fmt.Errorf("match glob error: %w", err)
		}
		return NewToken(m), nil
	default:
		return EmptyToken, fmt.Errorf("invalid match style: %s", parsedArgs["style"].String)
	}
}
*/

// --- helpers ---------------------------------------------------------------

func stringListFromToken(t *Token) ([]string, error) {
	lst, err := t.AsList()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(lst))
	for i := range lst {
		out[i] = lst[i].String
	}
	return out, nil
}

// runeFromToken: accept a one-rune string or an integer codepoint
func runeFromToken(t *Token) (rune, error) {
	// try integer first
	if iv, err := t.AsInt(); err == nil {
		return rune(iv), nil
	}
	s := t.String
	if s == "" {
		return 0, fmt.Errorf("expected rune, got empty string")
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		return 0, fmt.Errorf("invalid rune in %q", s)
	}
	return r, nil
}

// procFromToken ensures the token is invocable (Proc or Procer)
func procFromToken(t *Token) (Proc, error) {
	switch v := t.Data.(type) {
	case Proc:
		return v, nil
	case Procer:
		return v.Proc, nil
	default:
		return nil, fmt.Errorf("expected proc, got %T", t.Data)
	}
}

// callUnaryRuneProc: call adz proc with a single string (one-rune) argument, expect a bool
func callUnaryRuneProc(interp *Interp, p Proc, ch rune) (bool, error) {
	argTok := NewToken(string(ch))
	ret, err := p(interp, []*Token{NewToken("proc"), argTok})
	if err != nil {
		return false, err
	}
	// treat non-empty / true-ish as true
	b, berr := ret.AsBool()
	if berr == nil {
		return b, nil
	}
	return ret.String != "" && ret.String != "0", nil
}

func ProcStringsFormat(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("format", "the format specification; see go doc fmt"),
		ArgHelp("args", "the values with which to populate the returned string"),
	)
	as.Help = "format a string using provided values"
	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}

	valList, err := bound["args"].AsList()
	var values = make([]any, 0, len(valList))
	for i := range valList {
		if valList[i].Data != nil {
			values = append(values, valList[i].Data)
		} else {
			values = append(values, valList[i].String)
		}
	}
	return NewToken(fmt.Sprintf(bound["format"].String, values...)), nil
}

// --- predicates / contains --------------------------------------------------

func ProcStringsContains(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("s", "string to search"),
		ArgHelp("substr", "substring to find within s"),
	)
	as.Help = "Contains reports whether substr is within s."
	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	if strings.Contains(bound["s"].String, bound["substr"].String) {
		return TrueToken, nil
	}
	return FalseToken, nil
}

func ProcStringsContainsAny(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("s", "string to search"),
		ArgHelp("chars", "set of characters; reports true if any are in s"),
	)
	as.Help = "ContainsAny reports whether any Unicode code points in chars are within s."
	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	if strings.ContainsAny(bound["s"].String, bound["chars"].String) {
		return TrueToken, nil
	}
	return FalseToken, nil
}

func ProcStringsContainsRune(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("s", "string to search"),
		ArgHelp("r", "rune (one-char string or integer code point)"),
	)
	as.Help = "ContainsRune reports whether the rune r is within s."
	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	r, er := runeFromToken(bound["r"])
	if er != nil {
		return EmptyToken, er
	}
	if strings.ContainsRune(bound["s"].String, r) {
		return TrueToken, nil
	}
	return FalseToken, nil
}

func ProcStringsEqualFold(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("s", "first string"),
		ArgHelp("t", "second string"),
	)
	as.Help = "EqualFold reports whether s and t are equal under Unicode case-folding."
	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	if strings.EqualFold(bound["s"].String, bound["t"].String) {
		return TrueToken, nil
	}
	return FalseToken, nil
}

func ProcStringsHasPrefix(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("prefix", "prefix"))
	as.Help = "HasPrefix reports whether s begins with prefix."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	if strings.HasPrefix(b["s"].String, b["prefix"].String) {
		return TrueToken, nil
	}
	return FalseToken, nil
}

func ProcStringsHasSuffix(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("suffix", "suffix"))
	as.Help = "HasSuffix reports whether s ends with suffix."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	if strings.HasSuffix(b["s"].String, b["suffix"].String) {
		return TrueToken, nil
	}
	return FalseToken, nil
}

// --- count / index ----------------------------------------------------------

func ProcStringsCount(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("substr", "substring"))
	as.Help = "Count counts the number of non-overlapping instances of substr in s."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(fmt.Sprintf("%d", strings.Count(b["s"].String, b["substr"].String))), nil
}

func ProcStringsIndex(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("substr", "substring"))
	as.Help = "Index returns the index of the first instance of substr in s, or -1."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(fmt.Sprintf("%d", strings.Index(b["s"].String, b["substr"].String))), nil
}

func ProcStringsLastIndex(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("substr", "substring"))
	as.Help = "LastIndex returns the index of the last instance of substr in s, or -1."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(fmt.Sprintf("%d", strings.LastIndex(b["s"].String, b["substr"].String))), nil
}

func ProcStringsIndexAny(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("chars", "character set"))
	as.Help = "IndexAny returns the index of the first instance in s of any Unicode code points in chars, or -1."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(fmt.Sprintf("%d", strings.IndexAny(b["s"].String, b["chars"].String))), nil
}

func ProcStringsLastIndexAny(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("chars", "character set"))
	as.Help = "LastIndexAny returns the index of the last instance in s of any Unicode code points in chars, or -1."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(fmt.Sprintf("%d", strings.LastIndexAny(b["s"].String, b["chars"].String))), nil
}

func ProcStringsIndexRune(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("r", "rune"))
	as.Help = "IndexRune returns the index of the first instance of the rune r in s, or -1."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	r, er := runeFromToken(b["r"])
	if er != nil {
		return EmptyToken, er
	}
	return NewToken(fmt.Sprintf("%d", strings.IndexRune(b["s"].String, r))), nil
}

// --- split / join -----------------------------------------------------------

func ProcStringsSplit(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("sep", "separator"))
	as.Help = "Split slices s into all substrings separated by sep and returns a list."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	parts := strings.Split(b["s"].String, b["sep"].String)
	toks := make([]*Token, len(parts))
	for i := range parts {
		toks[i] = NewToken(parts[i])
	}
	return NewList(toks), nil
}

func ProcStringsSplitN(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("sep", "separator"), ArgHelp("n", "max splits"))
	as.Help = "SplitN slices s into substrings separated by sep and returns a list of at most n substrings."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	n, _ := b["n"].AsInt()
	parts := strings.SplitN(b["s"].String, b["sep"].String, n)
	toks := make([]*Token, len(parts))
	for i := range parts {
		toks[i] = NewToken(parts[i])
	}
	return NewList(toks), nil
}

func ProcStringsSplitAfter(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("sep", "separator"))
	as.Help = "SplitAfter slices s after each instance of sep and returns a list."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	parts := strings.SplitAfter(b["s"].String, b["sep"].String)
	toks := make([]*Token, len(parts))
	for i := range parts {
		toks[i] = NewToken(parts[i])
	}
	return NewList(toks), nil
}

func ProcStringsSplitAfterN(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("sep", "separator"), ArgHelp("n", "max splits"))
	as.Help = "SplitAfterN slices s after each instance of sep and returns at most n substrings."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	n, _ := b["n"].AsInt()
	parts := strings.SplitAfterN(b["s"].String, b["sep"].String, n)
	toks := make([]*Token, len(parts))
	for i := range parts {
		toks[i] = NewToken(parts[i])
	}
	return NewList(toks), nil
}

func ProcStringsJoin(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("elems", "list of strings"), ArgHelp("sep", "separator"))
	as.Help = "Join concatenates the elements of elems to create a single string separated by sep."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	elems, er := stringListFromToken(b["elems"])
	if er != nil {
		return EmptyToken, er
	}
	return NewToken(strings.Join(elems, b["sep"].String)), nil
}

// --- replace / repeat -------------------------------------------------------

func ProcStringsReplace(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("s", "string"),
		ArgHelp("old", "old substring"),
		ArgHelp("new", "replacement"),
		ArgHelp("n", "number of replacements (use -1 for all)"),
	)
	as.Help = "Replace returns a copy of s with the first n non-overlapping instances of old replaced by new."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	n, _ := b["n"].AsInt()
	return NewToken(strings.Replace(b["s"].String, b["old"].String, b["new"].String, n)), nil
}

func ProcStringsReplaceAll(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("old", "old substring"), ArgHelp("new", "replacement"))
	as.Help = "ReplaceAll returns a copy of s with all non-overlapping instances of old replaced by new."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.ReplaceAll(b["s"].String, b["old"].String, b["new"].String)), nil
}

func ProcStringsRepeat(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("s", "string"),
		&Argument{
			Name:   "count",
			Coerce: NewToken("int"),
			Help:   "repeat count",
		},
	)
	as.Help = "Repeat returns a new string consisting of count copies of s."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	n, _ := b["count"].AsInt()
	return NewToken(strings.Repeat(b["s"].String, n)), nil
}

// --- case folding -----------------------------------------------------------

func ProcStringsToLower(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"))
	as.Help = "ToLower returns s with all Unicode letters mapped to their lower case."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.ToLower(b["s"].String)), nil
}

func ProcStringsToUpper(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"))
	as.Help = "ToUpper returns s with all Unicode letters mapped to their upper case."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.ToUpper(b["s"].String)), nil
}

func ProcStringsToTitle(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"))
	as.Help = "ToTitle returns s with all Unicode letters mapped to their title case (Unicode upper, mostly)."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.ToTitle(b["s"].String)), nil
}

// --- trim -------------------------------------------------------------------

func ProcStringsTrim(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("cutset", "characters to trim"))
	as.Help = "Trim returns a slice of s with all leading and trailing Unicode code points in cutset removed."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.Trim(b["s"].String, b["cutset"].String)), nil
}
func ProcStringsTrimLeft(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("cutset", "characters to trim"))
	as.Help = "TrimLeft returns a slice of s with all leading Unicode code points in cutset removed."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.TrimLeft(b["s"].String, b["cutset"].String)), nil
}
func ProcStringsTrimRight(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("cutset", "characters to trim"))
	as.Help = "TrimRight returns a slice of s with all trailing Unicode code points in cutset removed."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.TrimRight(b["s"].String, b["cutset"].String)), nil
}
func ProcStringsTrimSpace(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"))
	as.Help = "TrimSpace returns s without leading and trailing white space as defined by Unicode."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.TrimSpace(b["s"].String)), nil
}
func ProcStringsTrimPrefix(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("prefix", "prefix"))
	as.Help = "TrimPrefix returns s without the provided leading prefix string; if s doesn't start with prefix, s is returned unchanged."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.TrimPrefix(b["s"].String, b["prefix"].String)), nil
}
func ProcStringsTrimSuffix(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("suffix", "suffix"))
	as.Help = "TrimSuffix returns s without the provided trailing suffix string; if s doesn't end with suffix, s is returned unchanged."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(strings.TrimSuffix(b["s"].String, b["suffix"].String)), nil
}

// --- compare ----------------------------------------------------------------

func ProcStringsCompare(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("a", "string"), ArgHelp("b", "string"))
	as.Help = "Compare returns an integer comparing two strings lexicographically: -1 if a < b, 0 if a == b, +1 if a > b."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	return NewToken(fmt.Sprintf("%d", strings.Compare(b["a"].String, b["b"].String))), nil
}

// --- callback-based ---------------------------------------------------------

// strings.Map: map each rune via a proc: mapper :: r -> string (or empty to drop)
func ProcStringsMap(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("mapper", "proc to map each rune"), ArgHelp("s", "string"))
	as.Help = "Map returns a copy of the string s with all its characters modified by the mapping function mapper(ch). Return empty string to drop a rune."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}

	mapperProc, perr := procFromToken(b["mapper"])
	if perr != nil {
		return EmptyToken, perr
	}

	mapped := strings.Map(func(r rune) rune {
		// Call mapper; if it returns empty, drop rune; if it returns a single rune, use it.
		ret, err := mapperProc(interp, []*Token{NewToken("mapper"), NewToken(string(r))})
		if err != nil {
			// On error, preserve original rune (defensive choice)
			return r
		}
		if ret.String == "" {
			return -1 // drop
		}
		rr, size := utf8.DecodeRuneInString(ret.String)
		if rr == utf8.RuneError && size == 1 {
			return r
		}
		return rr
	}, b["s"].String)

	return NewToken(mapped), nil
}

// strings.Fields: split on runs of space (no callback needed)
func ProcStringsFields(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"))
	as.Help = "Fields splits the string s around each instance of one or more consecutive white space characters."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	parts := strings.Fields(b["s"].String)
	toks := make([]*Token, len(parts))
	for i := range parts {
		toks[i] = NewToken(parts[i])
	}
	return NewList(toks), nil
}

// strings.FieldsFunc: predicate proc gets one rune; truthy => split
func ProcStringsFieldsFunc(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("s", "string"), ArgHelp("pred", "proc predicate on rune"))
	as.Help = "FieldsFunc splits the string s at each run of Unicode code points c satisfying pred(c)."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}

	pred, perr := procFromToken(b["pred"])
	if perr != nil {
		return EmptyToken, perr
	}

	splitter := func(r rune) bool {
		ok, err := callUnaryRuneProc(interp, pred, r)
		if err != nil {
			// fallback: treat errors as non-split
			return false
		}
		return ok
	}

	parts := strings.FieldsFunc(b["s"].String, splitter)
	toks := make([]*Token, len(parts))
	for i := range parts {
		toks[i] = NewToken(parts[i])
	}
	return NewList(toks), nil
}

// Bonus: quick Unicode helpers if you want them exposed too
func ProcUnicodeIsLetter(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String, ArgHelp("r", "rune"))
	as.Help = "Returns true if rune is a Unicode letter."
	b, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	r, er := runeFromToken(b["r"])
	if er != nil {
		return EmptyToken, er
	}
	if unicode.IsLetter(r) {
		return TrueToken, nil
	}
	return FalseToken, nil
}
