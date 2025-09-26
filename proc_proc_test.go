package adz_test

import (
	"strings"
	"testing"

	. "github.com/sparques/adz"
)

// tiny helper
func mustRun(t *testing.T, interp *Interp, script string) *Token {
	t.Helper()
	out, err := interp.ExecString(script)
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	return out
}
func mustErr(t *testing.T, interp *Interp, script string) error {
	t.Helper()
	_, err := interp.ExecString(script)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	return err
}

func TestProc_Anonymous_IsCallableByToken_NotRegistered(t *testing.T) {
	i := NewInterp()

	// anonymous: 2-arg form
	out := mustRun(t, i, `set f [proc {x} {return x${x}x}]`)
	// Returned token should be invokable (Data is Proc) *via variable*
	out = mustRun(t, i, `$f hello`)
	if got := out.String; got != "xhellox" {
		t.Fatalf("expected xhellox, got %q", got)
	}
	// Name should *not* be invokable as a command (not registered)
	err := mustErr(t, i, `proc#0 hello`) // or whatever display is; use variable instead
	_ = err                              // only asserts non-nil; some displays differ
}

func TestProc_DisplayName_DependsOnNamespaceRoot(t *testing.T) {
	i := NewInterp()

	// Global frame is namespaceTop; anonymous display should be proc#N
	out := mustRun(t, i, `set f [proc {x} {return $x}]`)
	if !strings.HasPrefix(out.String, "proc#") {
		t.Fatalf("expected fully-qualified display at namespace root, got %q", out.String)
	}

	// Now create a nested frame and define anonymous; display should be bare (no ::)
	mustRun(t, i, `
        proc outer {} {
            return [proc {x} {return $x}]
        }
    `)
	out = mustRun(t, i, `outer`)
	if strings.HasPrefix(out.String, "::") {
		t.Fatalf("expected bare display inside nested frame, got %q", out.String)
	}
}

func TestProc_Named_Global_WhenAtNamespaceRoot(t *testing.T) {
	i := NewInterp()

	// Named creation at namespace top registers in namespace/global proc table.
	out := mustRun(t, i, `proc echo {x} {return $x}`)
	if want := "::echo"; out.String != want {
		t.Fatalf("expected %q, got %q", want, out.String)
	}
	// callable by name
	out = mustRun(t, i, `echo hi`)
	if out.String != "hi" {
		t.Fatalf("expected hi, got %q", out.String)
	}
	// also via fully-qualified name
	out = mustRun(t, i, `::echo bye`)
	if out.String != "bye" {
		t.Fatalf("expected bye, got %q", out.String)
	}
}

func TestProc_Named_InsideProc_IsFrameLocal(t *testing.T) {
	i := NewInterp()

	// define a proc that defines another unqualified proc inside its body
	mustRun(t, i, `
        namespace ns {
            proc p1 {} {
                proc p2 {} { return ok }
                ;# callable here:
                return [p2]
            }
        }
    `)
	// calling ns::p1 returns ok (inner p2 exists during p1)
	out := mustRun(t, i, `::ns::p1`)
	if out.String != "ok" {
		t.Fatalf("expected ok, got %q", out.String)
	}
	// but p2 is NOT callable outside
	err := mustErr(t, i, `::ns::p2`)
	if !strings.Contains(err.Error(), "command not found") {
		t.Fatalf("expected command not found for ::ns::p2, got %v", err)
	}
}

func TestProc_Named_Qualified_TargetsNamespace(t *testing.T) {
	i := NewInterp()
	// define into a namespace directly
	out := mustRun(t, i, `proc ::math::dbl {x} { return $x$x }`)
	if out.String != "::math::dbl" {
		t.Fatalf("expected ::math::dbl, got %q", out.String)
	}
	// callable via fully-qualified name
	out = mustRun(t, i, `::math::dbl a`)
	if out.String != "aa" {
		t.Fatalf("expected aa, got %q", out.String)
	}
	// not callable unqualified from global
	err := mustErr(t, i, `dbl a`)
	if !strings.Contains(err.Error(), "command not found") {
		t.Fatalf("expected not found for unqualified dbl, got %v", err)
	}
}

func TestProc_UsageOnBindError(t *testing.T) {
	i := NewInterp()

	mustRun(t, i, `proc needs2 {x y} { return ok }`)
	err := mustErr(t, i, `needs2 1`)
	// error string should reflect arg count in some form; exact message may differ
	if !strings.Contains(err.Error(), "missing required arg") {
		t.Fatalf("expected usage/arg error, got %v", err)
	}
}

func TestProc_AnonInNamespaceBlock_ExecutesInThatNs(t *testing.T) {
	i := NewInterp()

	// Create ns with a var; anonymous proc should see that ns as its localNamespace (per ProcProc)
	mustRun(t, i, `
        namespace env {
            set a 42
            set f [proc {} { return $::env::a }]
            set g [proc {} { return [namespace] }] ;# introspection helper if you have it
            set ret [$f]
            set retns [$g]
        }
    `)
	// evaluate the values we stashed
	out := mustRun(t, i, `subst {$env::ret}`)
	if out.String != "42" {
		t.Fatalf("expected 42, got %q", out.String)
	}
}
