package adz

import (
	"bufio"
	"bytes"

	"github.com/sparques/adz/parser"
)

func LexString(str string) (Script, error) {
	return LexBytes([]byte(str))
}

func LexBytes(buf []byte) (Script, error) {
	lineScanner := bufio.NewScanner(bytes.NewBuffer(buf))
	lineScanner.Split(parser.LineSplit)
	script := make(Script, 0)
	for lineScanner.Scan() {
		// bytes.NewBuffer says we shouldn't use underlying bytes after this call, but
		// since we're only reading from it it should be ok... I think.
		cmd := make(Command, 0)
		tokScanner := bufio.NewScanner(bytes.NewBuffer(lineScanner.Bytes()))
		tokScanner.Split(parser.TokenSplit)
		for tokScanner.Scan() {
			cmd = append(cmd, NewTokenBytes(tokScanner.Bytes()))
		}
		// skip empty lines and comments
		if len(cmd) == 0 || cmd[0].String[0] == '#' {
			continue
		}

		script = append(script, cmd)
	}

	return script, nil
}

func LexBytesToList(buf []byte) (List, error) {
	list := make(List, 0)
	tokScanner := bufio.NewScanner(bytes.NewBuffer(buf))
	tokScanner.Split(parser.TokenSplit)
	for tokScanner.Scan() {
		tok := &Token{
			String: stripLiteralBrackets(tokScanner.Text()),
		}
		// tok := NewTokenString(tokScanner.Text())
		// fmt.Printf("Before: %s\nAfter: %s\n", tok.String, tok.Literal())
		// tok.String = tok.Literal()
		list = append(list, tok)
		// list = append(list, NewTokenString(tokScanner.Text()))
	}
	return list, nil
}

func LexStringToList(str string) ([]*Token, error) {
	return LexBytesToList([]byte(str))
}
