package adz

import (
	"strings"
	"testing"
)

type A struct {
	field string
}

func (a A) GetB() B {
	return B{}
}

type B struct{}

func (b B) GetA() A {
	return A{}
}

type wrapHarness struct {
	Name string
}

func (w *wrapHarness) Join(parts ...string) string {
	return strings.Join(parts, "|")
}

func (w *wrapHarness) Pair(a, b string) string {
	return a + ":" + b
}

func Test_Wrap(t *testing.T) {
	interp := NewInterp()

	var a A
	interp.SetVar("atok", Wrap(a))

	ret, err := interp.ExecString(`set btok [$atok GetB]; set atok [$btok GetA]`)
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	if _, ok := ret.Data.(Procer); !ok {
		t.Fatalf("expected wrapped callable result, got %T", ret.Data)
	}
}

func Test_Wrap_VariadicMethod(t *testing.T) {
	interp := NewInterp()
	interp.SetVar("obj", Wrap(&wrapHarness{}))

	ret, err := interp.ExecString(`$obj Join a b c`)
	if err != nil {
		t.Fatalf("variadic method call failed: %v", err)
	}
	if ret.String != "a|b|c" {
		t.Fatalf("got %q want %q", ret.String, "a|b|c")
	}
}

func Test_WrapObject_MethodSigDefaultsDriveInvocation(t *testing.T) {
	interp := NewInterp()
	sigs := map[string]*ArgSet{
		"Pair": NewArgSet("Pair",
			ArgHelp("a", "first value"),
			ArgDefaultHelp("b", NewToken("fallback"), "second value"),
		),
	}
	interp.SetVar("obj", WrapObject(&wrapHarness{}, sigs))

	ret, err := interp.ExecString(`$obj Pair left`)
	if err != nil {
		t.Fatalf("method call with defaulted arg failed: %v", err)
	}
	if ret.String != "left:fallback" {
		t.Fatalf("got %q want %q", ret.String, "left:fallback")
	}
}
