package adz

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// helpers
func runStr(t *testing.T, interp *Interp, script string) (*Token, error) {
	t.Helper()
	return interp.ExecString(script)
}
func mustStr(t *testing.T, tok *Token) string {
	t.Helper()
	if tok == nil {
		t.Fatal("nil token")
	}
	return tok.String
}
func newI() *Interp { return NewInterp() }

// -----------------------------
// Token basics
// -----------------------------
func Test_Token_Quoted_Literal_Summary(t *testing.T) {
	tok := NewTokenString(`a b`)
	if got := tok.Quoted(); got != "{a b}" {
		t.Fatalf("Quoted mismatch: %q", got)
	}
	if got := tok.Literal(); got != "a b" {
		t.Fatalf("Literal mismatch: %q", got)
	}
	long := NewTokenString("0123456789__MIDDLE__abcdefghi")
	sum := long.Summary()
	if !strings.HasPrefix(sum, "0123456789") || !strings.Contains(sum, "…") {
		t.Fatalf("Summary missing elision: %q", sum)
	}
}

func Test_Token_AsBool_Int_Float(t *testing.T) {
	b, err := NewTokenString("true").AsBool()
	if err != nil || b != true {
		t.Fatalf("AsBool true failed: %v %v", b, err)
	}
	i, err := NewTokenString("42").AsInt()
	if err != nil || i != 42 {
		t.Fatalf("AsInt 42 failed: %v %v", i, err)
	}
	f, err := NewTokenString("3.5").AsFloat()
	if err != nil || f != 3.5 {
		t.Fatalf("AsFloat 3.5 failed: %v %v", f, err)
	}
}

func Test_Token_AsList_Index_Slice(t *testing.T) {
	tok := NewTokenString("a b c d")
	list, err := tok.AsList()
	if err != nil {
		t.Fatalf("AsList failed: %v", err)
	}
	if len(list) != 4 || list[2].String != "c" {
		t.Fatalf("List content mismatch: %+v", list)
	}
	if got := tok.Index(3).String; got != "d" {
		t.Fatalf("Index mismatch: %s", got)
	}
	if got := tok.Index(-1).String; got != "d" {
		t.Fatalf("Index -1 mismatch: %s", got)
	}
	if got := tok.Slice(1, 2).String; got != "b c" {
		t.Fatalf("Slice 1..2 mismatch: %s", got)
	}
	// reverse slice when start > end
	if got := tok.Slice(3, 1).String; got != "d c b" {
		t.Fatalf("Reverse slice mismatch: %s", got)
	}
}

func Test_Token_IndexSet_Append(t *testing.T) {
	tok := NewTokenString("a b")
	nt, err := tok.IndexSet(3, NewTokenString("Z"))
	if err != nil {
		t.Fatalf("IndexSet err: %v", err)
	}
	// pads with empties up to index
	if nt.String != "a b {} Z" { // EmptyToken prints "" for missing slots
		t.Fatalf("IndexSet padded list mismatch: %q", nt.String)
	}
	// Append creates a new list token
	ap := nt.Append(NewTokenString("E"))
	if ap.String != "a b {} Z E" {
		t.Fatalf("Append mismatch: %q", ap.String)
	}
}

// -----------------------------
// Substitution engine
// -----------------------------
func Test_Subst_Literals_Quotes(t *testing.T) {
	interp := newI()

	tok, err := interp.Subst(NewTokenString(`{literal block}`))
	if err != nil || tok.String != "literal block" {
		t.Fatalf("brace literal subst failed: %v %q", err, tok)
	}
	tok, err = interp.Subst(NewTokenString(`"spaced text"`))
	if err != nil || tok.String != "spaced text" {
		t.Fatalf("quote literal subst failed: %v %q", err, tok)
	}
}

func Test_Subst_Variables_And_Escapes(t *testing.T) {
	interp := newI()
	_, _ = interp.SetVar("a", NewTokenString("A"))
	// whole-token var
	tok, err := interp.Subst(NewTokenString(`$a`))
	if err != nil || tok.String != "A" {
		t.Fatalf("whole-token var failed: %v %q", err, tok)
	}
	// embedded var + escapes
	tok, err = interp.Subst(NewTokenString(`pre-${a}-\t-\n-post`))
	if err != nil {
		t.Fatalf("subst err: %v", err)
	}
	if !strings.Contains(tok.String, "pre-A-") || !strings.Contains(tok.String, "\t") || !strings.Contains(tok.String, "\n") {
		t.Fatalf("escape/var mixing failed: %q", tok.String)
	}
	// unicode \u hex
	tok, err = interp.Subst(NewTokenString(`\u263A`)) // ☺
	if err != nil || tok.String != "☺" {
		t.Fatalf("unicode subst failed: %v %q", err, tok)
	}
}

func Test_Subst_Subcommands(t *testing.T) {
	interp := newI()
	// rely on 'int' existing and returning its arg coerced (present in StdLib)
	tok, err := interp.Subst(NewTokenString(`[int 5]`))
	if err != nil || tok.String != "5" {
		t.Fatalf("subcommand failed: %v %q", err, tok)
	}
	// empty [] is allowed and ignored
	tok, err = interp.Subst(NewTokenString(`pre[]post`))
	if err != nil || tok.String != "prepost" {
		t.Fatalf("empty subcommand handling failed: %v %q", err, tok)
	}
}

// -----------------------------
// Interpreter exec & procs
// -----------------------------
func Test_Exec_UnknownCommand_Wrapped(t *testing.T) {
	interp := newI()
	_, err := interp.ExecString(`nope`)
	if !errors.Is(err, ErrCommandNotFound) {
		t.Fatalf("expected ErrCommandNotFound, got %v", err)
	}
}

func Test_Exec_Procer_Dispatch_And_ErrorWrap(t *testing.T) {
	interp := newI()

	// install a Procer via Proc type and have it return an error to hit ErrCommand wrapping
	fail := Proc(func(_ *Interp, _ []*Token) (*Token, error) {
		return EmptyToken, fmt.Errorf("boom")
	})
	if err := interp.Proc("fail", fail); err != nil {
		t.Fatal(err)
	}
	_, err := interp.ExecString(`fail`)
	if err == nil || !strings.HasPrefix(err.Error(), "fail") {
		t.Fatalf("expected wrapped error with command name, got %v", err)
	}
}

func Test_Exec_Panic_Trapped(t *testing.T) {
	interp := newI()
	panicProc := Proc(func(_ *Interp, _ []*Token) (*Token, error) {
		panic("yikes")
	})
	if err := interp.Proc("kaboom", panicProc); err != nil {
		t.Fatal(err)
	}
	_, err := interp.ExecString(`kaboom`)
	if !errors.Is(err, ErrGoPanic) {
		t.Fatalf("expected ErrGoPanic, got %v", err)
	}
}

func Test_Exec_MaxCallDepth(t *testing.T) {
	interp := newI()
	interp.MaxCallDepth = 16

	var recur Proc
	recur = Proc(func(it *Interp, _ []*Token) (*Token, error) {
		// recurse until guard trips
		return it.ExecString(`recur`)
	})
	if err := interp.Proc("recur", recur); err != nil {
		t.Fatal(err)
	}
	_, err := interp.ExecString(`recur`)
	if !errors.Is(err, ErrMaxCallDepthExceeded) {
		t.Fatalf("expected ErrMaxCallDepthExceeded, got %v", err)
	}
}

// -----------------------------
// Variables & Ref (Getter/Setter/Deleter)
// -----------------------------
func Test_Vars_Local_Qualified_And_Ref(t *testing.T) {
	interp := newI()

	// local var
	_, _ = interp.SetVar("x", NewTokenString("local"))
	got, err := interp.GetVar("x")
	if err != nil || got.String != "local" {
		t.Fatalf("local var mismatch: %v %q", err, got)
	}
	// qualified var
	_, _ = interp.SetVar("::q", NewTokenString("qual"))
	got, err = interp.GetVar("::q")
	if err != nil || got.String != "qual" {
		t.Fatalf("qualified var mismatch: %v %q", err, got)
	}

	// cross-namespace ref using Ref Getter/Setter
	ns := NewNamespace("ns1")
	interp.Namespaces["ns1"] = ns
	ns.Vars["v"] = NewTokenString("base")
	ref := (&Ref{Namespace: ns, Name: "v"}).Token() // Data=Ref getter/setter
	// install alias 'alias' in local frame, points to ns1::v
	_, _ = interp.SetVar("alias", ref)

	// read-through
	rt, err := interp.GetVar("alias")
	if err != nil || rt.String != "base" {
		t.Fatalf("Ref get mismatch: %v %q", err, rt)
	}
	// write-through
	_, err = interp.SetVar("alias", NewTokenString("mut"))
	if err != nil {
		t.Fatalf("Ref set err: %v", err)
	}
	if ns.Vars["v"].String != "mut" {
		t.Fatalf("Ref did not propagate write")
	}
	// delete-through
	_, err = interp.DelVar("alias")
	if err != nil {
		t.Fatalf("Ref delete err: %v", err)
	}
	if _, ok := ns.Vars["v"]; ok {
		t.Fatalf("Ref Del should remove target")
	}
}

// -----------------------------
// Namespaces
// -----------------------------
func Test_Namespace_Proc_And_Resolution(t *testing.T) {
	interp := newI()

	// Query current ns
	tok, err := runStr(t, interp, `namespace`)
	if err != nil || tok.String != "::" {
		t.Fatalf("namespace query failed: %v %q", err, tok)
	}

	// Create a namespace and execute in it
	tok, err = runStr(t, interp, `namespace ::n1 { int 5 }`)
	if err != nil || tok.String != "5" {
		t.Fatalf("namespace exec failed: %v %q", err, tok)
	}

	// Qualified var in new ns, then read via qualified name
	_, err = runStr(t, interp, `namespace ::n1 { set _ {} }`) // push frame in ns
	if err != nil {
		t.Fatal(err)
	}
	// SetVar fully qualified (go-side) and read via ExecString
	_, _ = interp.SetVar("::n1::a", NewTokenString("42"))
	tok, err = runStr(t, interp, `subst {$::n1::a}`)
	if err != nil || tok.String != "42" {
		t.Fatalf("qualified var subst failed: %v %q", err, tok)
	}
}

func Test_ResolveProc_Order(t *testing.T) {
	interp := newI()

	// define proc "p" in global and in a child namespace; local should win when inside that ns
	global := Proc(func(_ *Interp, _ []*Token) (*Token, error) { return NewTokenString("G"), nil })
	ns := NewNamespace("X")
	interp.Namespaces["X"] = ns

	if err := interp.Proc("::p", global); err != nil {
		t.Fatal(err)
	}
	local := Proc(func(_ *Interp, _ []*Token) (*Token, error) { return NewTokenString("L"), nil })
	if err := interp.Proc("::X::p", local); err != nil {
		t.Fatal(err)
	}

	// from global, unqualified "p" should hit global
	tok, err := runStr(t, interp, `p`)
	if err != nil || tok.String != "G" {
		t.Fatalf("global p mismatch: %v %q", err, tok)
	}

	// in ::X, unqualified should hit local
	_, err = runStr(t, interp, `namespace ::X { }`)
	if err != nil {
		t.Fatal(err)
	}
	interp.Push(&Frame{localNamespace: ns, localVars: ns.Vars})
	tok, err = runStr(t, interp, `p`)
	interp.Pop()
	if err != nil || tok.String != "L" {
		t.Fatalf("local p mismatch: %v %q", err, tok)
	}

	// AbsoluteProc should retrieve by fully qualified id
	if got := interp.AbsoluteProc("::X::p"); got == nil {
		t.Fatalf("AbsoluteProc failed for ::X::p")
	}
}

// -----------------------------
// Core StdLib procs (types.go)
// -----------------------------
func Test_Std_True_False_Bool_Int_Float_Tuple_GoType(t *testing.T) {
	interp := newI()

	// true/false/bool
	tok, err := runStr(t, interp, `true`)
	if err != nil || tok.String != "true" {
		t.Fatalf("true failed: %v %q", err, tok)
	}
	tok, err = runStr(t, interp, `false`)
	if err != nil || tok.String != "false" {
		t.Fatalf("false failed: %v %q", err, tok)
	}
	tok, err = runStr(t, interp, `bool on`)
	if err != nil || tok.String != "on" {
		t.Fatalf("bool on failed: %v %q", err, tok)
	}

	// int/float
	tok, err = runStr(t, interp, `int 41`)
	if err != nil || tok.String != "41" {
		t.Fatalf("int failed: %v %q", err, tok)
	}
	tok, err = runStr(t, interp, `float 3.25`)
	if err != nil || tok.String != "3.25" {
		t.Fatalf("float failed: %v %q", err, tok)
	}

	// tuple
	tok, err = runStr(t, interp, `tuple {a b c} b`)
	if err != nil || tok.String != "b" {
		t.Fatalf("tuple ok failed: %v %q", err, tok)
	}
	_, err = runStr(t, interp, `tuple {a b c} z`)
	if err == nil {
		t.Fatalf("tuple should error on invalid choice")
	}

	// gotype
	type X struct{ N int }
	x := &X{N: 9}
	_, _ = interp.SetVar("v", NewToken(x))
	tok, err = runStr(t, interp, fmt.Sprintf(`gotype %T $v`, x))
	if err != nil || tok.String != fmt.Sprintf("%v", x) {
		t.Fatalf("gotype pass failed: %v %q", err, tok)
	}
	_, err = runStr(t, interp, `gotype *main.Y $v`)
	if err == nil {
		t.Fatalf("gotype should fail for incorrect type")
	}
}

// -----------------------------
// ExecScript line-wrapping for errors
// -----------------------------
func Test_ExecScript_LineNumber_Wrap(t *testing.T) {
	interp := newI()
	// line 0 ok, line 1 errors
	_, err := interp.ExecString("true\nnope\nfalse")
	if err == nil {
		t.Fatalf("expected error")
	}
	// Expect it to be wrapped as ErrLine on non-first-line failures
	if !strings.Contains(err.Error(), "line 1:") {
		t.Fatalf("expected line wrapping, got: %v", err)
	}
}
