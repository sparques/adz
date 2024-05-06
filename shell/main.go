package main

import (
	"adz"
	"adz/parser"
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	interp := adz.NewInterp()
	interp.Stdout = os.Stdout
	interp.SetVar("PROMPT", adz.NewTokenString("? "))

	lineScanner := bufio.NewScanner(os.Stdin)
	lineScanner.Split(parser.LineSplit)

	promptVar, _ := interp.GetVar("PROMPT")
	fmt.Fprint(os.Stdout, promptVar.String)
	for lineScanner.Scan() {
		script, err := adz.LexBytes(lineScanner.Bytes())
		if err != nil {
			showError(os.Stdout, err)
			continue
		}
		out, err := interp.ExecScript(script)
		if out != nil {
			fmt.Fprint(os.Stdout, out.String)
		}
		if err != nil {
			showError(os.Stdout, err)
		}

		promptVar, _ = interp.GetVar("PROMPT")
		fmt.Fprint(os.Stdout, promptVar.String)
	}
}

func showError(w io.Writer, err error) {
	fmt.Fprintf(w, "\x1b[31mError:\x1b[m %s\n", err)
}
