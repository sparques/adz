package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sparques/adz"
	adzmath "github.com/sparques/adz/math"
	"github.com/sparques/adz/repl"
)

const defaultPrompt = "\x1b\\[33m|> \x1b\\[m"

// wrap things together so we can actually exit
func main() {
	os.Exit(run())
}

func run() int {
	// instantiate our shell
	interp := newRDTInterp()

	// load debug namespace

	// if called with no arguments, start shell, otherwise concat arguments and treat as oneliner
	if len(os.Args) > 1 {
		script := strings.Join(os.Args[1:], " ")
		output, err := interp.ExecString(script)
		if err != nil {
			fmt.Printf("\x1b[1;31mError:\x1b[m %s\n\n", err)
			fmt.Println(output)
			return 1 // general error
		}
		fmt.Println(output)
		return 0
	}

	reader, restore, err := newCommandReader(os.Stdin, os.Stdout, interp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[1;31mError:\x1b[m %s\n", err)
		return 1
	}
	defer restore()

	for {
		cmd, err := reader.ReadCommand()
		if err == io.EOF {
			return 0
		}
		if err == repl.ErrInterrupted {
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "\x1b[1;31mError:\x1b[m %s\n", err)
			continue
		}

		output, err := interp.ExecScript(cmd.Script)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\x1b[1;31mError:\x1b[m %s\n", err)
		}
		if output != nil {
			fmt.Println(output.String)
		}
	}
}

type commandReader interface {
	ReadCommand() (*repl.Command, error)
}

func newCommandReader(in io.Reader, out io.Writer, interp *adz.Interp) (commandReader, func(), error) {
	file, ok := in.(interface {
		Fd() uintptr
		Stat() (os.FileInfo, error)
	})
	if !ok || !isTerminal(file) {
		return repl.NewReader(in), func() {}, nil
	}

	raw, err := repl.EnableRawMode(file)
	if err != nil {
		return nil, nil, err
	}

	editor := repl.NewLineEditor(in, out)
	reader := repl.NewInteractiveReader(editor, func(continued bool) string {
		if continued {
			return "> "
		}
		return prompt(interp)
	})

	return reader, func() {
		_ = raw.Restore()
	}, nil
}

type statter interface {
	Stat() (os.FileInfo, error)
}

func isTerminal(f statter) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func prompt(interp *adz.Interp) string {
	promptTok, err := interp.GetVar("PROMPT")
	if err != nil {
		promptTok = adz.NewTokenString(defaultPrompt)
		interp.SetVar("PROMPT", promptTok)
	}
	promptVal, err := interp.ExecLiteral(adz.List{adz.NewToken("subst"), promptTok})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error substituting PROMPT: %v\n", err)
		fmt.Fprintln(os.Stderr, "Reverting to default prompt.")
		promptTok = adz.NewTokenString("? ")
		interp.SetVar("PROMPT", promptTok)
		return promptTok.String
	}
	return promptVal.String
}

func newRDTInterp() *adz.Interp {
	interp := adz.NewInterp()
	// Load Debug stuff
	adz.LoadDebug(interp)
	interp.Stdout = os.Stdout
	interp.Stderr = os.Stderr
	interp.Stdin = os.Stdin
	// add commands here.
	adzmath.LoadProcs(interp)
	// shell stuff
	loadOS(interp)

	for _, kv := range os.Environ() {
		kvpair := strings.SplitN(kv, "=", 2)
		interp.SetVar(kvpair[0], adz.NewTokenString(kvpair[1]))
	}

	return interp
}
