package adz

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_Interp(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`set a 1; set b [set c $a$a$a]; return $a`)
	if err != ErrReturn {
		t.Errorf("Interp test: expected err to be return, got %s", err)
	}

	if out.String != "1" {
		t.Errorf("Interp test: expected out to be 1, got %s", out.String)
	}

	if interp.Vars["c"].String != "111" {
		t.Errorf("Interp test: expected c to be 111, got %s", interp.Vars["c"].String)
	}

}

func Test_Interp2(t *testing.T) {
	interp := NewInterp()

	_, err := interp.ExecString("set intlist [int 0 1 2 3 4 5]")
	if err != nil {
		t.Errorf("Interp2 test: expected err to be nil, got %s", err)
	}

	tok, err := interp.GetVar("intlist")
	if err != nil {
		t.Errorf("Interp2 test: expected err to be nil, got %s", err)
	}

	list, _ := tok.AsList()
	for i := range list {
		if i != list[i].Data.(int) {
			t.Errorf("Interp2 test: expected %d to be itself, got %d", i, list[i].Data.(int))

		}
	}
}

func Test_Interp3(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString("not [eq [int 0] [int 1]]")
	if err != nil {
		t.Errorf("Interp3 test: expected err to be nil, got %s", err)
	}

	if out != TrueToken {
		t.Errorf("Interp3 test: expected TrueToken, got %s", out.String)
	}

}

func Test_Interp4(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		set b [while true {
			set a [int 42]
			break [int 69]
			and this is okay, as long as we don't break parsing this can be anything
			Escapes \} work, too.
		}]
	`)

	if err != nil {
		t.Errorf("Interp4 test: expected err to be nil, got %s", err)
		return
	}

	if out.String != "69" {
		t.Errorf("Interp4 test: expected 69, got %s", out.String)
	}

	if b, err := interp.GetVar("b"); err != nil {
		t.Errorf("Interp4 test: expected b to be defined, got %s", err)
		return
	} else {
		if b.String != "69" || b.Data.(int) != 69 {
			t.Errorf("Interp4 test: expected b to be 69, got %s", b.String)
		}
	}

	if interp.Vars["a"].Data.(int) != 42 {
		t.Errorf("Interp4 test: expected a to be 42, got %s", interp.Vars["a"].String)
	}
}

func Test_Interp5(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		proc test {} {
			int 42
			
		}
		test
	`)
	if err != nil {
		t.Errorf("Interp5 test: expected err to be nil, got %s", err)
		return
	}

	if out.Data.(int) != 42 {
		t.Errorf("Interp5 test: expected 42, got %s", out.String)
	}

}

func Test_Interp6(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		proc test {} {
			return [int 42]
			and all this other junk
			really doesn't matter
			It does get parsed and slow everything down.
			*shrug*
		}
		test
	`)
	if err != nil {
		t.Errorf("Interp5 test: expected err to be nil, got %s", err)
		return
	}

	if out.Data.(int) != 42 {
		t.Errorf("Interp5 test: expected 42, got %s", out.String)
	}

}

func Test_Interp7(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		proc test {arg} {
			return $arg
		}
		test [int 42]
	`)
	if err != nil {
		t.Errorf("Interp5 test: expected err to be nil, got %s", err)
		return
	}

	if out.Data.(int) != 42 {
		t.Errorf("Interp5 test: expected 42, got %s", out.String)
	}
}

func Test_Interp8(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		set a [int 42]
		proc test {} {
			set a test
			test2
		}
		proc test2 {} {
			set a test2
			test3
		}
		proc test3 {} {
			set a test3
		}
		test
		list $a
	`)
	if err != nil {
		t.Errorf("Interp5 test: expected err to be nil, got %s", err)
		return
	}

	if i, ok := out.Data.(int); !ok || i != 42 {
		t.Errorf("Interp8 test: expected 42, got %s", out.String)
	}
}

func Test_Interp9(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		proc test {a b c d} {return $d}
		test _ _ _ [int 42]
	`)
	if err != nil {
		t.Errorf("Interp9 test: expected err to be nil, got %s", err)
		return
	}

	if i, ok := out.Data.(int); !ok || i != 42 {
		t.Errorf("Interp9 test: expected 42, got %s", out.String)
	}
}

func Test_Interp10(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		if {eq [int 42] 42} then {
			false
		}
	`)
	if err != nil {
		t.Errorf("Interp9 test: expected err to be nil, got %s", err)
		return
	}

	if i, ok := out.Data.(bool); !ok || i {
		t.Errorf("Interp9 test: expected bool false, got %s", out.String)
	}
}

func Test_HelloWorld(t *testing.T) {
	expected := "Hello World\nHello  World\nHello     World\\n\n"
	interp := NewInterp()
	buf := bytes.NewBuffer(nil)
	interp.Stdout = buf
	_, err := interp.ExecString(`
		print Hello       World\n
		print "Hello  World\n"
		println {Hello     World\n}
	`)
	if err != nil {
		t.Errorf("HelloWorld test: expected err to be nil, got %s", err)
		return
	}
	if buf.String() != expected {
		t.Errorf("expected %s, got %s", []byte(expected), []byte(buf.String()))
	}

}

func Test_Catch(t *testing.T) {
	interp := NewInterp()
	out, err := interp.ExecString(`
		set fail [catch {list poopy} ret err]
		list $fail $ret $err
	`)

	if err != nil {
		t.Errorf("catch test: expected err to be nil, got %s", err)
		return
	}

	fmt.Println(out.String)
}
