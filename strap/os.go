package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sparques/adz"
	"github.com/sparques/adz/repl"
)

func loadOS(interp *adz.Interp) {
	interp.Proc("exec", procExec)
	interp.Proc("cd", procCd)
	interp.Proc("pwd", procPwd)
	interp.Proc("source", procSource)
	interp.Proc("exit", procExit)
	interp.Proc("cat", procCat)
	interp.Proc("glob", procGlob)
}

// procExec implements exec command.
// exec /path/to/exe args...
func procExec(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	// TODO Use ArgSet, have flag to treat cmd's stdout as return value
	if len(args) < 2 {
		return adz.EmptyToken, adz.ErrArgMinimum(1)
	}

	cargs := make([]string, len(args)-1)
	for i, v := range args[1:] {
		cargs[i] = v.String
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, cargs[0], cargs[1:]...)

	cmd.Stdin = interp.Stdin
	cmd.Stdout = interp.Stdout
	cmd.Stderr = interp.Stderr

	err := cmd.Run()
	//cmd.Wait()
	// fmt.Println("waited")

	return adz.EmptyToken, err
}

// procCd implements the cd command
func procCd(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	if len(args) > 2 {
		return adz.EmptyToken, fmt.Errorf("cd: expected one or none arguments")
	}

	if len(args) == 1 {
		homedir := os.Getenv(`HOME`)
		// just one arg, change to home dir
		return adz.EmptyToken, os.Chdir("/" + homedir)
	}

	return adz.EmptyToken, os.Chdir(args[1].String)
}

func procPwd(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	dir, err := os.Getwd()
	return adz.NewTokenString(dir), err
}

// procSource takes one argument; it executes the contents of the file
func procSource(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	if len(args) != 2 {
		return adz.EmptyToken, adz.ErrArgCount(1, len(args)-1)
	}

	contents, err := os.ReadFile(args[1].String)
	if err != nil {
		return adz.EmptyToken, err
	}
	script, err := adz.LexBytes(contents)
	if err != nil {
		return adz.EmptyToken, err
	}

	return interp.ExecScript(script)
}

func procToJSON(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	// TODO arg count check
	// TODO options: json encode, indent encode, bson.ExtJson stuff
	if args[1].Data != nil {
		out, err := json.MarshalIndent(args[1].Data, "", "  ")
		if err != nil {
			return adz.EmptyToken, err
		}
		return adz.NewTokenString(string(out)), nil
	}

	// data was nil... is this json *data* encoded as a string?
	var out any
	err := json.Unmarshal([]byte(args[1].String), &out)
	if err == nil {
		tok := adz.NewTokenString(args[1].String)
		tok.Data = out
		return tok, nil
	}

	// do fancy stuff like check if it's a list to encode as a list
	// might need to add some TextMarshallers to adz.Token
	return adz.EmptyToken, nil
}

// TODO: procInfo returns information about strap
func procInfo(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	return adz.EmptyToken, nil
}

func procExit(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {

	code := 0
	if len(args) > 1 {
		code, _ = args[1].AsInt()
	}
	_ = repl.RestoreTerminal()
	os.Exit(code)
	return adz.EmptyToken, nil
}

// cat returns the contents of a file
func procCat(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	as := adz.NewArgSet(
		"cat",
		adz.ArgDefaultCoerce("-escape", adz.NewToken("none"), adz.NewToken("tuple {none jsonstr rdt}")),
		adz.Arg("file"),
	)
	parsedArgs, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return adz.EmptyToken, err
	}

	contents, err := os.ReadFile(parsedArgs["file"].String)
	if err != nil {
		return adz.EmptyToken, err
	}

	return adz.NewTokenString(string(contents)), nil
}

func procGlob(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	var files []*adz.Token
	for _, pattern := range args[1:] {
		m, err := filepath.Glob(pattern.String)
		if err != nil {
			return adz.EmptyToken, err
		}
		for i := range m {
			files = append(files, adz.NewTokenString(m[i]))
		}
	}
	return adz.NewList(files), nil
}

func procJson(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
	// parse argument as json data, return token with same json string, but save
	// the unmarshalled go object into the .Data
	if len(args) != 2 {
		return adz.EmptyToken, adz.ErrArgCount
	}

	var obj any
	err := json.Unmarshal([]byte(args[1].String), &obj)
	if err != nil {
		return adz.EmptyToken, fmt.Errorf("could not unmarshal json: %w", err)
	}

	args[1].Data = obj
	return args[1], nil
}
