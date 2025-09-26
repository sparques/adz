package adz

import (
	"strings"
	"testing"
)

func mustRun(t *testing.T, ip *Interp, script string) string {
	t.Helper()
	out, err := ip.ExecString(script)
	if err != nil {
		t.Fatalf("exec error for script:\n%s\nerr: %v", script, err)
	}
	if out == nil {
		return ""
	}
	return out.String
}

func runErr(t *testing.T, ip *Interp, script string) (string, error) {
	t.Helper()
	out, err := ip.ExecString(script)
	if out != nil {
		return out.String, err
	}
	return "", err
}

func TestNamespace_FullyQualifiedVarAlwaysWorks(t *testing.T) {
	ip := NewInterp()
	mustRun(t, ip, `set ::a 42`)
	got := mustRun(t, ip, `subst {$::a}`)
	if got != "42" {
		t.Fatalf("want 42, got %q", got)
	}
}

func TestNamespace_UnqualifiedVarInProc_IsLocalOnly(t *testing.T) {
	ip := NewInterp()
	mustRun(t, ip, `set ::a 42`)
	// $a should NOT see ::a when inside proc unless you explicitly import or qualify
	mustRun(t, ip, `proc ::p {} { return $a }`) // defining proc should succeed
	_, err := ip.ExecString(`p`)
	if err == nil || !strings.Contains(err.Error(), "no such variable") {
		t.Fatalf("expected no-such-variable error for unqualified $a in proc, got %v", err)
	}
	// sanity: fully qualified inside proc works
	mustRun(t, ip, `proc ::q {} { return $::a }`)
	got := mustRun(t, ip, `q`)
	if got != "42" {
		t.Fatalf("want 42 from $::a, got %q", got)
	}
}

func TestNamespace_DefineAndCallWithinNamespace(t *testing.T) {
	ip := NewInterp()
	// Define proc a inside ns1. It creates unqualified proc b during its body.
	mustRun(t, ip, `namespace ::ns1 { proc a {} { proc [namespace]::b {} {}; return ::ns1::b } }`)
	// Calling ::ns1::a should return the FQN of b
	got := mustRun(t, ip, `::ns1::a`)
	if got != "::ns1::b" {
		t.Fatalf("want ::ns1::b, got %q", got)
	}
	// b is NOT global
	_, err := ip.ExecString(`b`)
	if err == nil || !strings.Contains(err.Error(), "command not found") {
		t.Fatalf("expected command-not-found for global b, got %v", err)
	}
	// but namespaced b exists and is runnable
	if _, err := ip.ExecString(`::ns1::b`); err != nil {
		t.Fatalf("expected ::ns1::b to be callable, got %v", err)
	}
}

func TestNamespace_IntrospectionReturnsCurrentNamespace(t *testing.T) {
	ip := NewInterp()
	// In a namespaced block, [namespace] should be the FQN of that namespace.
	got := mustRun(t, ip, `namespace ::nsX { namespace }`)
	if got != "::nsX" {
		t.Fatalf("want ::nsX from [namespace], got %q", got)
	}
	// Global shell likely returns :: (adjust if your introspection prints empty)
	got2 := mustRun(t, ip, `namespace`)
	if got2 != "::" && got2 != "" { // accept either :: or "" as global marker
		t.Fatalf("want :: or empty for global, got %q", got2)
	}
}

func TestImport_ReadsAndOptionallyWritesThrough(t *testing.T) {
	ip := NewInterp()
	mustRun(t, ip, `set ::c 41`)
	// read via import
	mustRun(t, ip, `proc ::reader {} { import -var {{::c b}}; return $b }`)
	got := mustRun(t, ip, `reader`)
	if got != "41" {
		t.Fatalf("want 41 from imported c, got %q", got)
	}

	// write-through: depending on your branch, import may be RW now.
	// We'll assert write-through if allowed; otherwise assert an error appears.
	mustRun(t, ip, `proc ::writer {} { import -var {{::c c}}; set c 42; return $::c }`)
	got2, err := ip.ExecString(`writer`)
	if err == nil {
		if got2.String != "42" {
			t.Fatalf("import write-through path: want 42, got %q", got2.String)
		}
	} else {
		// read-only import path: writing to alias should fail
		if !strings.Contains(err.Error(), "cannot assign") && !strings.Contains(err.Error(), "read-only") {
			t.Fatalf("expected read-only/assign error, got %v", err)
		}
		// and original should remain 41
		got3 := mustRun(t, ip, `return $::c`)
		if got3 != "41" {
			t.Fatalf("read-only path: expected ::c unchanged (41), got %q", got3)
		}
	}
}

func TestNestedProc_DoesNotLeakLocalsAcrossFrames(t *testing.T) {
	ip := NewInterp()
	mustRun(t, ip, `
		proc ::outer {} {
			set x outer
			proc ::inner {} { return $x } ;# inner should NOT see $x unless imported
			return ::inner
		}
	`)
	// outer returns the name of inner
	got := mustRun(t, ip, `outer`)
	if got != "::inner" {
		t.Fatalf("want ::inner, got %q", got)
	}
	_, err := ip.ExecString(`inner`)
	if err == nil || !strings.Contains(err.Error(), "no such variable") {
		t.Fatalf("inner should not see outer's locals, got %v", err)
	}
}

func TestCallDepthReturnsToZeroAfterFailure(t *testing.T) {
	ip := NewInterp()
	// provoke an error in a nested call and make sure depth unwinds
	mustRun(t, ip, `proc ::bomb {} { return $nope }`)
	_, err := ip.ExecString(`bomb`)
	if err == nil {
		t.Fatalf("expected error")
	}
	if d := ip.CallDepth(); d != 0 {
		t.Fatalf("call depth leak: want 0, got %d", d)
	}
	// interpreter should still work after
	got := mustRun(t, ip, `set ::ok 1; subst {$::ok}`)
	if got != "1" {
		t.Fatalf("expected interpreter to keep working, got %q", got)
	}
}

// benchmark
func Benchmark_Interp1(b *testing.B) {
	interp := NewInterp()
	setup := `
		for {set i 0} {< $i 1000} {incr i} {
			proc proc$i {} {
				proc[+ $i 1]
			}
		}
		proc proc1000 {} {}
	`
	script := `for {set i 0} {< $i 1000} {incr i} {list} {
		proc$i
	}`
	interp.ExecString(setup)
	for b.Loop() {
		interp.ExecString(script)
	}
}
