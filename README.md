[![Coverage Status](https://coveralls.io/repos/github/sparques/adz/badge.svg?branch=master)](https://coveralls.io/github/sparques/adz?branch=master)
[![Go ReportCard](https://goreportcard.com/badge/sparques/adz)](https://goreportcard.com/report/sparques/adz)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/sparques/adz)


# ADZ

ADZ or adzlang is a Tcl-like scripting language. Similar in purpose to Tcl's original purpose, the idea is to provide an easy-to-bind scripting langauge **and shell** for Golang.

The general language parsing/lexing rules of Tcl applies to ADZ. Where ADZ differs is the implementation of the built-in commands and a few parsing corner cases.

ADZ does not in anyway aim to be compatible with Tcl. Tcl's ruleset and general idea is just easy to adapt and reuse.

Below is the [Octologue from Tcl](https://wiki.tcl-lang.org/page/Dodekalogue), modified to be true for ADZ. 

# Example: Adding a Command to ADZ

Initializing and adding a command to an ADZ interpreter is easy.

Currently by default the full base standard library of commands is added when an ADZ interpretter is initialized.

```
  interp := adz.NewInterp()
  interp.Procs = make(map[string]adz.Proc)
  interp.Procs["foo"] = func(interp *adz.Interp, args []*adz.Token) (*adz.Token, error) {
  	return adz.NewTokenString("bar"), nil
  }
```

The above golang code initializes an ADZ interpreter, removes all the base commands, then sets the command "foo" to the anonymous func which will always return bar.

```
	out, err := interp.ExecString(`foo`)
	fmt.Println(out.String, err)
```

The above will print "bar <nil>".

# Octologue

## Script
A script is composed of commands delimited by newlines or semicolons, and a command is composed of words delimited by whitespace.

## Evaluation
Substitutions in each word are processed, and the first word of each command is used to locate a routine, which is then called with the remaining words as its arguments.

ADZ does not (yet) support {*} argument expansion.

## Comment
If # is encountered when the name of a command is expected, it and the subsequent characters up to the next newline are ignored, except that a newline preceded by an odd number of \ characters does not terminate the comment.

## $varname
Replaced by the value of the variable named varname, which is a sequence of one or more letters, digits, or underscore characters, or namespace separators. If varname is enclosed in braces, it is composed of all the characters between the braces.

## \char
Replaced with the character following char, except when char is a, b, f, n, t, r, v, which respectively signify the unicode characters audible alert (7), backspace (8), form feed (c), newline (a), carriage return (d), tab (9), and vertical tab (b), respectively. Additionally, when char is x or u, it represents a unicode character by 2 hexadecimal digits, or 1 or more hexadecimal digits, respectively.

## Brackets
A script may be embedded at any position of a word by enclosing it in brackets. The embedded script is passed verbatim to the interpreter for execution and the result is inserted in place of the script and its enclosing brackets. The result of a script is the result of the last routine in the script.

## Quotes
In a word enclosed in quotes, whitespace and semicolons have no special meaning.

## Braces
In a word enclosed in braces, whitespace, semicolons, $, brackets, and \ have no special meaning, except that \newline substitution is performed when the newline character is preceded by an odd number of \ characters. Any nested opening brace character must be paired with a subsequent matching closing brace character. An opening or closing brace character preceded by an odd number of \ characters is not used to pair braces.


# Intended Uses

## MCUs

Currently, my main target for ADZ is as a commandline shell for an RP2040 based pocket-computer. Something akin to DOS. 

For this reason I'm trying to keep memory usage really low. Currently the main underlying storage are strings and I may have to rework the whole code base and change over to byte slices for memory efficiency purposes--golang passes strings by value, so it doesn't take much for many many copies of the same string to be in memory. For the most part strings are passed around within a struct that is referenced by a pointer and this helps limit memory use. That said, since running on a microcontroller is the main goal, all other considerations are secondary. 

## Distributed, Interruptable Interpreter
I'd also like to use ADZ as glue-logic for passing around "scripts" between microservices. This isn't as begging-for-RCE as it sounds. With a little addtional work, the ADZ interpreter can be fully serialized and saved to disk, sent over the wire, or be backed by a database like badgerdb. 

Since the interpreter is a set of builtin commands, text-based procedures, and text-based variables, serializing the interpreter is relatively easy, as long as you don't want to do it while a command is running. 


# Limitations
## Performance

Not great. ADZ is meant to be a good basic shell and easy to mix with golang. But it's not going to beat... probably any other language. Even Tcl itself JIT compiles to byte code and achieves remarkably good performance.

## Debugging

The implementation has been purposely kept very simple and naïve, so luxuries like knowing what line an error happened are not available. Really, if you're making a BIG program in ADZ, you're using it wrong.

## Documentation

What documentation?


# Future Improvements

## Namespaces

While this will probably never be incorporated for the MCU version of adz, as it adds too much overhead, for other usecases, having support for namespaces would be very helpful

## Packages

Following on namespaces, Packages both go-based and ADZ based would be useful. 
  