package adz

import (
	"errors"
	"fmt"
)

func init() {
	StdLib["pipeline"] = ProcPipeline
}

// With 1 argument, chain executes it as a script.
// After each command in the script, the output of the command is saved into
// the special variable $|. This can then be used in the next command
// as input. With two arguments the final output of the chain is stored
// in a variable of that name.
//
// Example:
//
//		pipeline result {
//			curl wobpage.net/data
//			lindex $| 0
//			field $| key.value.*
//			touch $|
//	 }
func ProcPipeline(interp *Interp, args []*Token) (*Token, error) {
	var (
		script Script
		result *Token = EmptyToken
		save   *Token
		err    error
	)
	switch len(args) {
	case 2:
		script, err = args[1].AsScript()
		if err != nil {
			return EmptyToken, fmt.Errorf("arg 0: could not parse as script: %w", err)
		}
	case 3:
		save = args[1]
		script, err = args[2].AsScript()
		if err != nil {
			return EmptyToken, fmt.Errorf("arg 1: could not parse as script: %w", err)
		}
	default:
		return EmptyToken, ErrArgCount
	}
	if len(args) == 2 {
		// todo
	}

	for _, cmd := range script {
		result, err = interp.Exec(cmd)
		switch {
		case err == nil: // no error? fine.
		case errors.Is(err, ErrBreak): // got ErrBreak; skip to the rest of the script
			break
		default:
			return EmptyToken, err
		}
		interp.SetVar("|", result)
	}
	interp.DelVar("|")
	if save != nil {
		interp.SetVar(save.String, result)
	}

	return result, nil
}
