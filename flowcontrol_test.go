package adz

import (
	"errors"
	"strings"
	"testing"
)

func Test_FlowControl_IfElseIfElse(t *testing.T) {
	ip := NewInterp()

	got := mustRun(t, ip, `if false {set x no} elseif true then {set x yes} else {set x later}`)
	if got != "yes" {
		t.Fatalf("want yes, got %q", got)
	}

	got = mustRun(t, ip, `if false {set x no} elseif false {set x nope} else {set x else}`)
	if got != "else" {
		t.Fatalf("want else, got %q", got)
	}
}

func Test_FlowControl_IfMalformedDoesNotExecuteBody(t *testing.T) {
	ip := NewInterp()

	_, err := ip.ExecString(`if true {set x ok} nonsense`)
	if err == nil {
		t.Fatalf("expected syntax error")
	}
	if !strings.Contains(err.Error(), "expected") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := ip.GetVar("x"); !errors.Is(err, ErrNoVar) {
		t.Fatalf("malformed if should not have executed body; got err=%v", err)
	}
}

func Test_FlowControl_DoWhileContinueStillEvaluatesCondition(t *testing.T) {
	ip := NewInterp()

	got := mustRun(t, ip, `
		set i 0
		do {
			incr i
			continue
		} while {< $i 3}
		return $i
	`)
	if got != "3" {
		t.Fatalf("want 3, got %q", got)
	}
}

func Test_FlowControl_ForEachEmptyVarListErrors(t *testing.T) {
	ip := NewInterp()

	_, err := ip.ExecString(`foreach {} {a b} {set x nope}`)
	if err == nil {
		t.Fatalf("expected empty var list to fail")
	}
	if !strings.Contains(err.Error(), "missing required arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_FlowControl_CatchArityAndVars(t *testing.T) {
	ip := NewInterp()

	_, err := ip.ExecString(`catch`)
	if err == nil {
		t.Fatalf("expected catch with no script to fail")
	}

	got := mustRun(t, ip, `
		catch {throw boom} out err
		list $out $err
	`)
	if !strings.HasPrefix(got, "{} ") || !strings.Contains(got, "boom") {
		t.Fatalf("expected empty result and boom-bearing error, got %q", got)
	}

	got = mustRun(t, ip, `
		catch {set x ok} out err
		list $out $err
	`)
	if got != "ok {}" {
		t.Fatalf("want %q, got %q", "ok {}", got)
	}
}

func Test_FlowControl_BreakContinueReturnRejectExtraArgs(t *testing.T) {
	ip := NewInterp()

	cases := []string{
		`break a b`,
		`continue a b`,
		`return a b`,
	}

	for _, script := range cases {
		_, err := ip.ExecString(script)
		if err == nil {
			t.Fatalf("%s should fail", script)
		}
		if !strings.Contains(err.Error(), "wrong number of args") {
			t.Fatalf("%s: unexpected error: %v", script, err)
		}
	}
}
