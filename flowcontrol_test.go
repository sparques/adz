package adz

import "testing"

func Test_While(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
	set a 10
	set b [false]
	set c 0
	while {not [== $a 0]} {
		set a [+ $a -1]
		if $b {set b [false]; continue}
		set b [true]
		set c [+ $c 1]
	}
	return $c
	`)
	if err != ErrReturn {
		t.Errorf("While test: expected err to be return, got %s", err)
	}

	if out.String != "5" {
		t.Errorf("While test: expected out to be 0, got %s", out.String)
	}

}
