package adz

import (
	"strings"
	"testing"
)

func tok(s string) *Token { return &Token{String: s} }

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func eqPos(t *testing.T, got []*Token, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("pos len: got %d want %d; got=%v", len(got), len(want), projPos(got))
	}
	for i := range want {
		if got[i].String != want[i] {
			t.Fatalf("pos[%d]: got %q want %q; full=%v", i, got[i].String, want[i], projPos(got))
		}
	}
}

func eqNamed(t *testing.T, got map[string]*Token, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("named len: got %d want %d; got=%v", len(got), len(want), projNamed(got))
	}
	for k, v := range want {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("named missing key %q; got=%v", k, projNamed(got))
		}
		if gv.String != v {
			t.Fatalf("named[%q]: got %q want %q", k, gv.String, v)
		}
	}
}

func projPos(ts []*Token) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.String
	}
	return out
}
func projNamed(m map[string]*Token) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.String
	}
	return out
}

func TestParseArgs_CommandOnly(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{})
	eqPos(t, p, []string{})
}

func TestParseArgs_PositionalOnly(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("a"), tok("b")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{})
	eqPos(t, p, []string{"a", "b"})
}

func TestParseArgs_SingleOrZeroCharAlwaysPositional(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-"), tok(""), tok("x")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{})
	eqPos(t, p, []string{"-", "", "x"})
}

func TestParseArgs_NamedSimplePairs(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-x"), tok("1"), tok("-y"), tok("2")})
	mustNoErr(t, err)
	eqPos(t, p, []string{})
	eqNamed(t, n, map[string]string{"-x": "1", "-y": "2"})
}

func TestParseArgs_MixedNamedAndPositional(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-x"), tok("1"), tok("b"), tok("-y"), tok("2")})
	mustNoErr(t, err)
	eqPos(t, p, []string{"b"})
	eqNamed(t, n, map[string]string{"-x": "1", "-y": "2"})
}

func TestParseArgs_MissingValueForNamedIsError(t *testing.T) {
	_, _, err := ParseArgs([]*Token{tok("cmd"), tok("-x")})
	if err == nil {
		t.Fatalf("expected error for missing value after -x")
	}
	// be generous on the exact text; just ensure it mentions the flag
	if !strings.Contains(err.Error(), "-x") {
		t.Fatalf("error should mention flag: %v", err)
	}
}

func TestParseArgs_DoubleDashStopsNamedParsing(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("--"), tok("-x"), tok("1"), tok("b")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{})
	eqPos(t, p, []string{"-x", "1", "b"})
}

func TestParseArgs_DoubleDashAtEnd_OK(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("a"), tok("--")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{})
	eqPos(t, p, []string{"a"})
}

func TestParseArgs_ValueMayBeDoubleDash(t *testing.T) {
	// The special meaning of "--" applies only when it's the current arg,
	// not when it's used as the value for a named arg.
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-x"), tok("--")})
	mustNoErr(t, err)
	eqPos(t, p, []string{})
	eqNamed(t, n, map[string]string{"-x": "--"})
}

func TestParseArgs_SingleDashAsFirstPositionalThenNamed(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-"), tok("-x"), tok("1")})
	mustNoErr(t, err)
	eqPos(t, p, []string{"-"})
	eqNamed(t, n, map[string]string{"-x": "1"})
}

func TestParseArgs_CommandPlusDashDashOnly(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("--")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{})
	eqPos(t, p, []string{})
}

func TestParseArgs_NamedThenDashDashThenMore(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-a"), tok("1"), tok("--"), tok("-b"), tok("2")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{"-a": "1"})
	eqPos(t, p, []string{"-b", "2"})
}

func TestParseArgs_PositionalBeforeAndAfterNamed(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("p1"), tok("-a"), tok("x"), tok("p2")})
	mustNoErr(t, err)
	eqNamed(t, n, map[string]string{"-a": "x"})
	eqPos(t, p, []string{"p1", "p2"})
}

func TestParseArgs_MultipleNamedPairsSequential(t *testing.T) {
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-a"), tok("1"), tok("-b"), tok("2"), tok("-c"), tok("3")})
	mustNoErr(t, err)
	eqPos(t, p, []string{})
	eqNamed(t, n, map[string]string{"-a": "1", "-b": "2", "-c": "3"})
}

func TestParseArgs_NamedValueIsAnotherFlagToken(t *testing.T) {
	// The value immediately following a named arg is taken as its value,
	// even if it looks like a flag; only the current token position is checked for "-".
	n, p, err := ParseArgs([]*Token{tok("cmd"), tok("-a"), tok("-b"), tok("-c"), tok("v")})
	mustNoErr(t, err)
	eqPos(t, p, []string{}) // because -b is the value for -a; -c is a new named
	eqNamed(t, n, map[string]string{"-a": "-b", "-c": "v"})
}
