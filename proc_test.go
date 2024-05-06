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
