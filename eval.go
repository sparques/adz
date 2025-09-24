package adz

func init() {
	StdLib["eval"] = ProcEval
}

func ProcEval(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum
	}
	return interp.ExecString(TokenJoin(args[1:], " "))
}
