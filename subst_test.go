package adz

import (
	"testing"
)

var SubstTest = [][2]string{
	[2]string{``, ""},
	[2]string{`"Hello World!"`, "Hello World!"},
	[2]string{`\x01\x02\x03\x04`, "\x01\x02\x03\x04"},
	[2]string{`\\\\`, `\\`},
	[2]string{`\x4dcd`, "\x4dcd"},
	// 5
	[2]string{`\0\a\b\r\f\n`, "\x00\a\b\r\f\n"},
	[2]string{`superduper isn't it?`, "superduper isn't it?"},
	[2]string{`{yes}`, "yes"},
	[2]string{`a[]`, "a"},
	[2]string{"[][][]", ""},
	// 10
	[2]string{"[]", ""},
	[2]string{"[foo]", "bar"},
	[2]string{"[foo and this gets ignored]", "bar"},
	[2]string{"utilize different path[foo and this gets ignored]", "utilize different pathbar"},
	[2]string{`\u1f20`, "\u1f20"},
	// 15
	[2]string{`$varname`, "varvalue"},
	[2]string{`${varname}`, "varvalue"},
	[2]string{`$varname$varname`, "varvaluevarvalue"},
	[2]string{`${varname}${varname}`, "varvaluevarvalue"},
	//[2]string{`${varname`, "varvalue"}, // this throws an error now
	[2]string{`{}`, ""},
	// 20
	[2]string{`"no space escape needed"`, "no space escape needed"},
	[2]string{`{no space escape needed}`, "no space escape needed"},
	[2]string{`but\ space\ escapes\ work\ too`, "but space escapes work too"},
}

func Test_Subst(t *testing.T) {
	for i, pair := range SubstTest {
		str := NewTokenString(pair[0])

		interp := NewInterp()
		interp.Proc("foo", func(*Interp, []*Token) (*Token, error) { return NewTokenString("bar"), nil })
		interp.SetVar("varname", NewTokenString("varvalue"))
		out, err := interp.Subst(str)

		if out == nil {
			t.Errorf("Subst test %d: interp returned a nil Token ptr, wtf?", i)
			continue
		}

		if err != nil {
			t.Errorf("Subst test %d: Expected Subst() to return nil error, got %s", i, err)
			continue
		}

		if out.String != pair[1] {
			t.Errorf("Subst test %d: expected %v, got %v / %s", i, []byte(pair[1]), []byte(out.String), out.String)
		}
	}
}

type getVarEndIndexTest struct {
	input  string
	output int
}

var getVarEndIndexTests = []getVarEndIndexTest{
	// 0
	getVarEndIndexTest{"$a", 2},
	getVarEndIndexTest{"$asdf", 5},
	getVarEndIndexTest{"$asdf$asdf", 5},
	getVarEndIndexTest{"$asdf asdf", 5},
	getVarEndIndexTest{"${asdf} asdf", 7},
	// 5
	getVarEndIndexTest{"${asdf asdf} asdf", 12},
	getVarEndIndexTest{"${asdf[asdf}", 12},
}

func Test_getVarEndIndex(t *testing.T) {
	for i, tc := range getVarEndIndexTests {
		res := getVarEndIndex(tc.input)
		if res != tc.output {
			t.Errorf("getVarEndIndex test %d, expected %d, got %d", i, tc.output, res)
		}
	}
}
