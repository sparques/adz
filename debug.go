package adz

import (
	"runtime/debug"
)

var DebugProcs = make(map[string]Proc)

func init() {
	DebugProcs["stack"] = ProcDebugStack
}

func LoadDebug(interp *Interp) {
	interp.LoadProcs("debug", DebugProcs)
}

func ProcDebugStack(interp *Interp, args []*Token) (*Token, error) {
	return NewToken(string(debug.Stack())), nil
}
