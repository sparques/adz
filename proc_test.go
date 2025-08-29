package adz

import (
	"testing"
)

/*
func Test_ParseArgs(t *testing.T) {
	argTok := NewTokenString("arg1   -named\tnamedval1 arg2 -named2 namedval2 arg3   --     -arg4 -arg5 arg6")
	args, _ := argTok.AsList()

	named, pos, err := ParseArgs(args)

	if err != nil {
		t.Errorf("expected nil err, got %s", err)
	}

	if named["named"].String != "namedval1" || named["named2"].String != "namedval2" {
		t.Errorf("named values not as expected")
	}

	if len(named) != 2 {
		t.Errorf("got wrong number of named args, expected 2, got %d", len(named))
	}

	expectedPos := []string{"arg1", "arg2", "arg3", "-arg4", "-arg5", "arg6"}

	for i := range pos {
		if pos[i].String != expectedPos[i] {
			t.Errorf("positional arg wrong, expected %s, got %s", expectedPos[i], pos[i].String)
		}
	}

}
*/

func Test_AnonProc(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		[proc _ {a b} {+ $a $b}] 38 4
	`)

	if err != nil {
		t.Errorf("expected nil err, got %s", err)
	}

	if out == nil || out.String != "42" {
		t.Errorf("expected 42, got %s", out.String)
	}
}

func Test_Macro(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		macro m1 {int 42}
		m1
	`)

	if err != nil {
		t.Errorf("expected nil err, got %s", err)
	}

	if out == nil || out.String != "42" {
		t.Errorf("expected 42, got %s", out.String)
	}
}

func Test_Macro2(t *testing.T) {
	interp := NewInterp()

	out, err := interp.ExecString(`
		[macro m1 {int 42}]
	`)

	if err != nil {
		t.Errorf("expected nil err, got %s", err)
	}

	if out == nil || out.String != "42" {
		t.Errorf("expected 42, got %s", out.String)
	}
}

type ArgTest struct {
	desc        string
	script      string
	expectedErr *string
	expectedOut string
}

func newString(str string) *string {
	return &str
}

var ArgTests = []ArgTest{
	{
		desc:        `simple case, accept no args`,
		script:      `proc test {} {}; test a`,
		expectedErr: newString("line 1: test: expected 0 args, got 1"),
		expectedOut: ``,
	},
	{
		desc:        `fail minimum of 1 arg`,
		script:      `proc test {1 args} {}; test`,
		expectedErr: newString("line 1: test: expected at least 1 args, got 0"),
		expectedOut: ``,
	},
	{
		desc:        `with {-args args}, anything goes`,
		script:      `proc test {args -args} {sort [var]}; test`,
		expectedErr: nil,
		expectedOut: `{args {}}`,
	},
	{
		desc:        `with {-args args}, anything goes`,
		script:      `proc test {args -args} {sort [var]}; test 1 2 3`,
		expectedErr: nil,
		expectedOut: `{args {1 2 3}}`,
	},
	{
		desc:        `with {-args args}, anything goes`,
		script:      `proc test {args -args} {sort [var]}; test -1 a -2 b -3 c`,
		expectedErr: nil,
		expectedOut: `{1 a} {2 b} {3 c} {args {}}`,
	},
	{
		desc:        `with {-args args}, anything goes`,
		script:      `proc test {args -args} {sort [var]}; test -1 a -2 b -3 c one two three`,
		expectedErr: nil,
		expectedOut: `{1 a} {2 b} {3 c} {args {one two three}}`,
	},
	{
		desc:        `with {-args}, any named arg is okay, but so help me if there's a positional arg!`,
		script:      `proc test {-args} {sort [var]}; test -1 a -2 b -3 c`,
		expectedErr: nil,
		expectedOut: `{1 a} {2 b} {3 c}`,
	},
	{
		desc:        `with {-args}, any named arg is okay, but so help me if there's a positional arg!`,
		script:      `proc test {-args} {sort [var]}; test -1 a -2 b -3 c NO`,
		expectedErr: newString("line 1: test: expected 0 args, got 1"),
		expectedOut: ``,
	},
	{
		desc:        `with {-args -required}, any named arg is okay, but we need -required`,
		script:      `proc test {-args -required} {sort [var]}; test -1 a -2 b -3 c -required {totes ok}`,
		expectedErr: nil,
		expectedOut: "{1 a} {2 b} {3 c} {required {totes ok}}",
	},
	{
		desc:        `with {-args -required}, any named arg is okay, but we need -required`,
		script:      `proc test {-args -required} {sort [var]}; test -1 a -2 b -3 c`,
		expectedErr: newString(`line 1: test: missing required arg -required`),
		expectedOut: "",
	},
}

func Test_Args(t *testing.T) {

	for i, tc := range ArgTests {
		interp := NewInterp()
		out, err := interp.ExecString(tc.script)
		if err != nil {
			if tc.expectedErr == nil {
				t.Errorf("Args test %d: expected err to be nil, got %s", i, err.Error())
				continue
			}
			if *tc.expectedErr != err.Error() {
				t.Errorf("Args test %d: expected err to be %s, got %s", i, *tc.expectedErr, err.Error())
			}
		}

		if out.String != tc.expectedOut {
			t.Errorf("Args test %d: expected out to be %s, got %s", i, tc.expectedOut, out.String)
		}
	}
}
