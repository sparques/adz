package adz

import "testing"

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
