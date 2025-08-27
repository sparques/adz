package adz

import (
	"fmt"
	"path/filepath"
	"strings"
)

func init() {
	StdLib["match"] = procMatch
}

func procMatch(interp *Interp, args []*Token) (*Token, error) {
	namedProto, posProto, err := ParseProto(NewToken(`{-style glob} {-matchcase true} pattern str`))
	if err != nil {
		return EmptyToken, err
	}

	parsedArgs, err := ParseArgs(namedProto, posProto, args[1:])
	if err != nil {
		return EmptyToken, err
	}

	pattern, str := parsedArgs["pattern"].String, parsedArgs["str"].String
	if !parsedArgs["matchcase"].IsTrue() {
		pattern = strings.ToLower(pattern)
		str = strings.ToLower(str)
	}

	switch parsedArgs["style"].String {
	case "substr":
		return NewToken(strings.Contains(str, pattern)), nil
	case "regex":
		return EmptyToken, ErrNotImplemented("regex")
	case "glob":
		m, err := filepath.Match(pattern, str)
		if err != nil {
			return EmptyToken, fmt.Errorf("match glob error: %w", err)
		}
		return NewToken(m), nil
	default:
		return EmptyToken, fmt.Errorf("invalid match style: %s", parsedArgs["style"].String)
	}
}
