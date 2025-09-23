package adz

import (
	"errors"
	"fmt"
)

func init() {
	StdLib["pipeline"] = ProcPipeline
	StdLib["->"] = ProcPipeline
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
	as := NewArgSet(args[0].String)
	as.Help = "pipeline (also has alias ->) evaluates {script} as a script. After each top-level command is ran, the result is saved into the variable |; so the result is accessible with '$|' in the next command. This makes long chains of commands using the output of one command as the input of the next much easier to read and write. The return result of the pipeline is the return value of the final command. If {result} is specified, the final result of the pipeline will be saved to a variable of that name."
	resultArg := ArgHelp("result", "variable name to save final result to")
	scriptArg := ArgHelp("script", "the script to run as a pipeline.")
	as.ArgGroups = []*ArgGroup{
		NewArgGroup(scriptArg),
		NewArgGroup(resultArg, scriptArg),
	}

	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	var (
		script Script
		result *Token = EmptyToken
	)

	script, err = bound["script"].AsScript()
	if err != nil {
		return EmptyToken, fmt.Errorf("could not parse {script} as script: %w", err)
	}
	for _, cmd := range script {
		result, err = interp.Exec(cmd)
		switch {
		case err == nil: // no error? fine.
		case errors.Is(err, ErrBreak): // got ErrBreak; skip over the rest of the script
			break
		default:
			return EmptyToken, err
		}
		interp.SetVar("|", result)
	}
	interp.DelVar("|")
	if bound["result"] != nil && bound["result"].String != "" {
		interp.SetVar(bound["result"].String, result)
	}

	return result, nil
}
