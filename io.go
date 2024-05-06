package adz

import "fmt"

func init() {
	StdLib["print"] = ProcPrint
	StdLib["println"] = ProcPrint
}

func ProcPrint(interp *Interp, args []*Token) (*Token, error) {
	fmt.Fprint(interp.Stdout, TokenJoin(args[1:], " "))
	if args[0].String[len(args[0].String)-1] == 'n' {
		fmt.Fprintf(interp.Stdout, "\n")
	}
	return EmptyToken, nil
}
