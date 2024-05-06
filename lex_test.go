package adz

import (
	"testing"
)

func Test_LexString(t *testing.T) {
	s := `
cmd1 arg1_1 ;# this comment will be skipped
  # so will this one
cmd2 arg2_1 arg2_2 arg_2_3\
arg_2_4
cmd3
# this is a comment that will be skipped and so will the empty line below this

cmd4 arg4_1;cmd5 arg5_1
# escaping a newline of a comment causes the comment \
to continue to the next line like this.
`
	expected := [][]string{
		[]string{`cmd1`, `arg1_1`},
		[]string{`cmd2`, `arg2_1`, `arg2_2`, "arg_2_3\\\narg_2_4"},
		[]string{`cmd3`},
		[]string{`cmd4`, `arg4_1`},
		[]string{`cmd5`, `arg5_1`},
	}
	out, _ := LexString(s)
	for l, cmd := range out {
		for ti, tok := range cmd {
			if expected[l][ti] != tok.String {
				t.Errorf("expected %s, got %s", expected[l][ti], tok.String)
			}
		}
	}
}
