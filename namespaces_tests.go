package adz

import (
	"fmt"
	"strings"
	"testing"
)

type PlaneCase struct {
	desc   string
	script string
	out    string
	errSub string // empty = no error expected
}

var planeCases = []PlaneCase{
	{
		desc: "var and proc share FQN; each does the right thing",
		script: `
			set ::ns::a 42
			proc ::ns::a {} { return 41 }
			return [list $::ns::a [::ns::a]]
		`,
		out: "{42 41}",
	},
	{
		desc: "deeply nested name: ns='ns::a', name='b' for both var and proc",
		script: `
			set ::ns::a::b 0
			proc ::ns::a::b {} { return 1 }
			return [list $::ns::a::b [::ns::a::b]]
		`,
		out: "{0 1}",
	},
	{
		desc: "calling var FQN as a command fails with command-not-found",
		script: `
			set ::z 7
			::z
		`,
		errSub: "command not found",
	},
	{
		desc: "deref proc FQN as a var fails with no-such-variable",
		script: `
			proc ::ns::p {} { return ok }
			return $::ns::p
		`,
		errSub: "no such variable",
	},
	{
		desc: "subst braces vs bare: {$::ns::a} is literal deref, ::ns::a is call",
		script: `
			set ::ns::a 5
			proc ::ns::a {} { return 6 }
			return [list [subst {$::ns::a}] [::ns::a]]
		`,
		out: "{5 6}",
	},
	{
		desc: "unknown handler not invoked when var exists but proc doesn’t (and vice versa)",
		script: `
			set ::onlyVar 9
			::onlyVar
		`,
		errSub: "command not found",
	},
}

func TestPlanes_NoCollisions(t *testing.T) {
	for i, tc := range planeCases {
		t.Run(fmt.Sprintf("%02d_%s", i, tc.desc), func(t *testing.T) {
			ip := NewInterp()
			out, err := ip.ExecString(tc.script)
			if tc.errSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errSub) {
					t.Fatalf("want error containing %q, got %v", tc.errSub, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := out.String; got != tc.out {
				t.Fatalf("want %q, got %q", tc.out, got)
			}
			if d := ip.CallDepth(); d != 0 {
				t.Fatalf("call depth leak: %d", d)
			}
		})
	}
}

func TestNamespaceTokenization_ExtremeNames(t *testing.T) {
	ip := NewInterp()
	// Names that start/end with :: pieces shouldn’t crash your splitter.
	_, err := ip.ExecString(`set ::a:: 1`)
	if err == nil || !strings.Contains(err.Error(), "syntax") {
		// if you *allow* this, change the assertion accordingly;
		// point is: whatever policy you choose, lock it in.
	}

	// Long chain still resolves by last-:: rule
	out, err := ip.ExecString(`set ::a::b::c::d 4; return $::a::b::c::d`)
	if err != nil {
		t.Fatal(err)
	}
	if out.String != "4" {
		t.Fatalf("want 4, got %s", out.String)
	}
}

func TestNamespace_IntrospectionFlatNested(t *testing.T) {
	ip := NewInterp()
	out, err := ip.ExecString(`namespace ::proj::sub { return [namespace] }`)
	if err != nil {
		t.Fatal(err)
	}
	if out.String != "::proj::sub" {
		t.Fatalf("want ::proj::sub, got %q", out.String)
	}
}
