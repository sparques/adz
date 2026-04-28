package adz

import (
	"fmt"
	"testing"
)

func Test_Math_MixedNumericOps(t *testing.T) {
	interp := newI()

	cases := []struct {
		script string
		want   string
	}{
		{script: `+ 1 2 3`, want: "6"},
		{script: `+ 1 2.5 3`, want: "6.5"},
		{script: `- 10 2.5 3`, want: "4.5"},
		{script: `* 2 2.5 3`, want: "15"},
		{script: `/ 9 2`, want: "4.5"},
		{script: `/ 9 2.0`, want: "4.5"},
	}

	for _, tc := range cases {
		tok, err := runStr(t, interp, tc.script)
		if err != nil {
			t.Fatalf("%s failed: %v", tc.script, err)
		}
		if tok.String != tc.want {
			t.Fatalf("%s: got %q want %q", tc.script, tok.String, tc.want)
		}
	}
}

func Test_Math_MixedNumericComparisons(t *testing.T) {
	interp := newI()

	cases := []struct {
		script string
		want   string
	}{
		{script: `< 1 1.5`, want: "true"},
		{script: `<= 2 2.0`, want: "true"},
		{script: `> 2.5 2`, want: "true"},
		{script: `>= 2.0 3`, want: "false"},
	}

	for _, tc := range cases {
		tok, err := runStr(t, interp, tc.script)
		if err != nil {
			t.Fatalf("%s failed: %v", tc.script, err)
		}
		if tok.String != tc.want {
			t.Fatalf("%s: got %q want %q", tc.script, tok.String, tc.want)
		}
	}
}

func Test_Math_Incr_MixedNumeric(t *testing.T) {
	interp := newI()

	if _, err := runStr(t, interp, `set i 1`); err != nil {
		t.Fatalf("set i failed: %v", err)
	}
	tok, err := runStr(t, interp, `incr i 2.5`)
	if err != nil {
		t.Fatalf("incr i 2.5 failed: %v", err)
	}
	if tok.String != "3.5" {
		t.Fatalf("incr i 2.5: got %q want %q", tok.String, "3.5")
	}

	if _, err := runStr(t, interp, `set f 1.5`); err != nil {
		t.Fatalf("set f failed: %v", err)
	}
	tok, err = runStr(t, interp, `incr f`)
	if err != nil {
		t.Fatalf("incr f failed: %v", err)
	}
	if tok.String != "2.5" {
		t.Fatalf("incr f: got %q want %q", tok.String, "2.5")
	}
}

func Test_Math_BitwiseOps(t *testing.T) {
	interp := newI()

	cases := []struct {
		script string
		want   string
	}{
		{script: `& 7 3`, want: "3"},
		{script: `| 4 1`, want: "5"},
		{script: `^ 6 3`, want: "5"},
		{script: `bitnot 6`, want: fmt.Sprint(^6)},
		{script: `&^ 15 6`, want: "9"},
		{script: `<< 3 2`, want: "12"},
		{script: `>> 16 2`, want: "4"},
	}

	for _, tc := range cases {
		tok, err := runStr(t, interp, tc.script)
		if err != nil {
			t.Fatalf("%s failed: %v", tc.script, err)
		}
		if tok.String != tc.want {
			t.Fatalf("%s: got %q want %q", tc.script, tok.String, tc.want)
		}
	}
}

func Test_Math_BitwiseRejectsFloat(t *testing.T) {
	interp := newI()

	cases := []string{
		`& 7 2.0`,
		`| 1.5 1`,
		`^ 3 1.5`,
		`bitnot 2.5`,
		`<< 3 1.0`,
		`>> 8 1.5`,
	}

	for _, script := range cases {
		if _, err := runStr(t, interp, script); err == nil {
			t.Fatalf("%s should fail for non-integer input", script)
		}
	}
}
