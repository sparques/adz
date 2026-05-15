package adz

import (
	"github.com/sparques/adz/parser"
)

func LexString(str string) (Script, error) {
	return LexBytes([]byte(str))
}

func LexBytes(buf []byte) (Script, error) {
	script := make(Script, 0, 1)
	for len(buf) > 0 {
		advance, line, err := parser.LineSplit(buf, true)
		if err != nil {
			return nil, err
		}
		if advance == 0 {
			break
		}
		buf = buf[advance:]

		cmd := make(Command, 0, 4)
		for len(line) > 0 {
			advance, token, err := parser.TokenSplit(line, true)
			if err != nil {
				return nil, err
			}
			if advance == 0 {
				break
			}
			line = line[advance:]
			if token != nil {
				cmd = append(cmd, NewTokenBytes(token))
			}
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
	list := make(List, 0, 4)
	for len(buf) > 0 {
		advance, token, err := parser.TokenSplit(buf, true)
		if err != nil {
			return nil, err
		}
		if advance == 0 {
			break
		}
		buf = buf[advance:]
		if token == nil {
			continue
		}
		tok := &Token{
			String: stripLiteralBrackets(string(token)),
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
