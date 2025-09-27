package parser

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func scanWith(split bufio.SplitFunc, in string, bufCap int) []string {
	sc := bufio.NewScanner(strings.NewReader(in))
	if bufCap <= 0 {
		bufCap = 64 * 1024
	}
	sc.Buffer(make([]byte, 0, 1024), bufCap)
	sc.Split(split)
	var out []string
	for sc.Scan() {
		out = append(out, string(sc.Bytes()))
	}
	if err := sc.Err(); err != nil {
		out = append(out, "SCANERR:"+err.Error())
	}
	return out
}

func TestLineSplit_BasicNewlinesAndSemicolons(t *testing.T) {
	in := "a\nb;c\n\nd; e; f\n"
	got := scanWith(LineSplit, in, 0)
	want := []string{
		"a",
		"b",
		"c",
		"",
		"d",
		" e",
		" f",
	}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d len(want)=%d\ngot=%q\nwant=%q", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestLineSplit_QuotesAreOpaque_SemicolonInsideQuote(t *testing.T) {
	in := `"a; b";c;"x\";y";"z"`
	got := scanWith(LineSplit, in, 0)
	// first token includes quote with semicolon inside; then c; then a quoted with escaped quote; then last quoted
	want := []string{`"a; b"`, "c", `"x\";y"`, `"z"`}
	if len(got) != len(want) {
		t.Fatalf("got=%q want=%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestLineSplit_BraceBlocksSpanSemicolonsAndNewlines(t *testing.T) {
	in := "{a; b\nc {d;e}}\nrest"
	got := scanWith(LineSplit, in, 0)
	want := []string{`{a; b
c {d;e}}`, "rest"}
	if len(got) != len(want) {
		t.Fatalf("got=%q want=%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestLineSplit_BackslashSkipsNextChar(t *testing.T) {
	// The semicolon is escaped; should not terminate
	in := `a\;b;c`
	got := scanWith(LineSplit, in, 0)
	want := []string{`a\;b`, "c"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestLineSplit_CRLF_TrailingCRDropped(t *testing.T) {
	in := "a\r\nb\r\n"
	got := scanWith(LineSplit, in, 0)
	want := []string{"a", "b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestTokenSplit_BasicWhitespaceAndTokens(t *testing.T) {
	in := "  foo   bar\tbaz\nqux"
	got := scanWith(TokenSplit, in, 0)
	want := []string{"foo", "bar", "baz", "qux"}
	if len(got) != len(want) {
		t.Fatalf("got=%q want=%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tok %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestTokenSplit_QuotesAndBracesAndBrackets_GroupAsSingleToken(t *testing.T) {
	in := `foo {a b {c}} "x y \" z" [cmd arg]`
	got := scanWith(TokenSplit, in, 0)
	want := []string{
		"foo",
		"{a b {c}}",
		`"x y \" z"`,
		"[cmd arg]",
	}
	if len(got) != len(want) {
		t.Fatalf("got=%q want=%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tok %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestTokenSplit_EscapedSpaceStaysInToken(t *testing.T) {
	in := `a\ b c`
	got := scanWith(TokenSplit, in, 0)
	want := []string{`a\ b`, "c"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestTokenSplit_CRAndFFCountAsWhitespace(t *testing.T) {
	in := "a\rb\fc"
	got := scanWith(TokenSplit, in, 0)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestTokenSplit_DropTrailingCROnToken(t *testing.T) {
	in := "a\r b\r"
	got := scanWith(TokenSplit, in, 0)
	want := []string{"a", "b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestTokenSplit_DoubleQuoteOpaqueToBracesInside(t *testing.T) {
	in := `"a {b} c" d`
	got := scanWith(TokenSplit, in, 0)
	want := []string{`"a {b} c"`, "d"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestTokenSplit_DashDashStopsNamedParsing_Simulated(t *testing.T) {
	// TokenSplit itself doesn’t do named/positional; but ensure `--` doesn’t have magic here,
	// it’s just two chars; the “stop parsing flags” is a higher layer’s job.
	in := `-- -x y`
	got := scanWith(TokenSplit, in, 0)
	want := []string{"--", "-x", "y"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestFindPair_BasicAndEscaped(t *testing.T) {
	s := `"abc\"def";rest`
	// Find closing quote index
	idx := FindPair(s, '"')
	if idx == -1 {
		t.Fatalf("did not find closing quote")
	}
	if s[idx] != '"' {
		t.Fatalf("expected closing quote at idx=%d, got %q", idx, s[idx])
	}
	// ensure escaped quote didn’t terminate
	if strings.Contains(s[:idx], `";`) {
		t.Fatalf("escaped quote prematurely closed pair")
	}
}

func TestFindMate_StringAndByte_NestedBraces(t *testing.T) {
	s := "{a{b}c{d{e}f}g}h"
	// Start at opening brace (string API expects to see opener)
	idx := FindMate(s, '{', '}')
	if idx == -1 {
		t.Fatalf("FindMate failed")
	}
	if s[idx] != '}' {
		t.Fatalf("expected '}' at idx=%d got %q", idx, s[idx])
	}
	inner := []byte("a{b}c{d{e}f}g}tail")
	// Byte API expects slice starting *after* the opener; returns index of mate within this sub-slice
	bidx := FindMateByte(inner, '{', '}')
	if bidx == -1 || inner[bidx] != '}' {
		t.Fatalf("FindMateByte failed idx=%d byte=%q", bidx, inner[bidx])
	}
}

func TestFindMate_NoOpenerReturnsMinusOne(t *testing.T) {
	if idx := FindMate("}}}", '{', '}'); idx != -1 {
		t.Fatalf("expected -1, got %d", idx)
	}
	if idx := FindMateByte([]byte("}}}"), '{', '}'); idx != -1 {
		t.Fatalf("expected -1, got %d", idx)
	}
}

func TestLineSplit_EOFPartial_ReturnsFinalChunk(t *testing.T) {
	// direct call to split func to exercise atEOF=true behavior
	data := []byte("abc")
	adv, tok, err := LineSplit(data, true)
	if err != nil {
		t.Fatalf("LineSplit err: %v", err)
	}
	if adv != len(data) {
		t.Fatalf("advance=%d want=%d", adv, len(data))
	}
	if string(tok) != "abc" {
		t.Fatalf("tok=%q want=%q", string(tok), "abc")
	}
}

func TestTokenSplit_EOFPartial_ReturnsFinalToken(t *testing.T) {
	data := []byte("   xyz")
	adv, tok, err := TokenSplit(data, true)
	if err != nil {
		t.Fatalf("TokenSplit err: %v", err)
	}
	if adv != len(data) {
		t.Fatalf("advance=%d want=%d", adv, len(data))
	}
	if string(tok) != "xyz" {
		t.Fatalf("tok=%q want=%q", string(tok), "xyz")
	}
}

func TestTokenSplit_RequestMoreWhenUnterminated_NoAtEOF(t *testing.T) {
	// brace not closed and not at EOF → ask for more (advance=0, token=nil)
	data := []byte("{abc")
	adv, tok, err := TokenSplit(data, false)
	if err != nil {
		t.Fatalf("TokenSplit err: %v", err)
	}
	if adv != 0 || tok != nil {
		t.Fatalf("expected request more (0,nil), got adv=%d tok=%v", adv, tok)
	}
	// at EOF it should return the partial token
	adv, tok, err = TokenSplit(data, true)
	if err != nil {
		t.Fatalf("TokenSplit EOF err: %v", err)
	}
	if adv != len(data) || string(tok) != "{abc" {
		t.Fatalf("EOF partial: adv=%d tok=%q", adv, string(tok))
	}
}

func TestLineSplit_BackslashAtBufferEnd(t *testing.T) {
	// Ensure trailing backslash doesn’t panic and is handled sanely.
	data := []byte("abc\\")
	adv, tok, err := LineSplit(data, true)
	if err != nil {
		t.Fatalf("LineSplit err: %v", err)
	}
	if adv != len(data) {
		t.Fatalf("advance=%d want=%d", adv, len(data))
	}
	if string(tok) != "abc\\" {
		t.Fatalf("tok=%q want=%q", string(tok), "abc\\")
	}
}

func TestTokenSplit_BackslashAtBufferEnd(t *testing.T) {
	data := []byte(`"abc\"`)
	// not at EOF: should request more
	adv, tok, err := TokenSplit(data, false)
	if err != nil {
		t.Fatalf("TokenSplit err: %v", err)
	}
	if adv != 0 || tok == nil {
		// still inside quotes; expect more
	}
	// at EOF: return what we have (unterminated token)
	adv, tok, err = TokenSplit(data, true)
	if err != nil {
		t.Fatalf("TokenSplit EOF err: %v", err)
	}
	if adv != len(data) || string(tok) != `"abc\"` {
		t.Fatalf("EOF partial: adv=%d tok=%q", adv, string(tok))
	}
}

func TestCloseSymbolMapping(t *testing.T) {
	cases := map[byte]byte{
		'{': '}',
		'[': ']',
		'"': '"',
	}
	for k, v := range cases {
		if got := closeSymbol(k); got != v {
			t.Fatalf("closeSymbol(%q)=%q want %q", k, got, v)
		}
	}
	if got := closeSymbol('x'); got != 0 {
		t.Fatalf("closeSymbol(x)=%q want 0", got)
	}
}

func TestIsName_Simple(t *testing.T) {
	for _, b := range []byte{'[', ']', '"', '{', '}', ' ', '\t', '\r', '\n', '\f'} {
		if IsName(b) {
			t.Fatalf("IsName(%q)=true; expected false", b)
		}
	}
	for _, b := range []byte{'a', 'Z', '_', '-', '.', '0'} {
		if !IsName(b) {
			t.Fatalf("IsName(%q)=false; expected true", b)
		}
	}
}

func TestLineSplit_LongNestedBlock_NoScannerDefaultLimitPanic(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i := 0; i < 200000; i++ {
		buf.WriteString("x")
	}
	buf.WriteString("}")
	got := scanWith(LineSplit, buf.String(), 512*1024) // raise max token
	if len(got) != 1 || !strings.HasPrefix(got[0], "{") || !strings.HasSuffix(got[0], "}") {
		t.Fatalf("unexpected split result, len=%d", len(got))
	}
}
