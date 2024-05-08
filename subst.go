package adz

import (
	"adz/parser"
	"fmt"
	"strconv"
	"strings"
)

func (interp *Interp) Subst(tok *Token) (*Token, error) {

	// DON'T MODIFY tok !!!
	switch {
	case len(tok.String) <= 1:
		return tok, nil
	case tok.String[0] == '{' && parser.FindMate(tok.String, '{', '}') == len(tok.String)-1:
		// we have a literal, remove brackets and return
		return NewTokenString(tok.String[1 : len(tok.String)-1]), nil
	case tok.String[0] == '"' && parser.FindPair(tok.String, '"') == len(tok.String)-1:
		// strip off quotes and otherwise do normal substitution
		tok = NewTokenString(tok.String[1 : len(tok.String)-1])
	case !strings.ContainsAny(tok.String, `[$\`):
		// token has no special characters in it, it's just a string and no further substitution is required
		return tok, nil
	case tok.String[0] == '[' && parser.FindMate(tok.String, '[', ']') == len(tok.String)-1:
		// the whole token is a subcommand, strip off braces and run as script
		return interp.ExecString(tok.String[1 : len(tok.String)-1])
	case tok.String[0] == '$' && getVarEndIndex(tok.String) == len(tok.String):
		// whole token is a variable; return the reverence variable
		return interp.GetVar(parseVarName(tok.String))
	}

	str := strings.Builder{}
	var mIdx int
	for i := 0; i < len(tok.String); i++ {
		switch tok.String[i] {
		case '\\':
			i++ // move past the inital backslash
			// bound check / corner case. We'll let a final backslash be a literal backslash--not sure how this
			// could get past the lexer/parser, though.
			if i == len(tok.String) {
				str.WriteByte('\\')
				break
			}
			switch tok.String[i] {
			case '0':
				str.WriteByte(0)
			case 'a':
				str.WriteByte(0x07)
			case 'b':
				str.WriteByte(0x08)
			case 't':
				str.WriteByte(0x09)
			case 'n':
				str.WriteByte(0x0A)
			case 'v':
				str.WriteByte(0x0B)
			case 'f':
				str.WriteByte(0x0C)
			case 'r':
				str.WriteByte(0x0D)
			case '\\':
				str.WriteByte(0x5C)
			case 'u':
				i++
				mIdx = 0
				for i+mIdx < len(tok.String) && isHex(tok.String[i+mIdx]) {
					mIdx++
				}
				if mIdx == 0 {
					continue
				}
				// oooh so confident
				hex, _ := strconv.ParseUint(tok.String[i:i+mIdx], 16, 0)
				str.WriteRune(rune(hex))
				i += mIdx
				continue
			case 'x':
				i++
				mIdx = 0
				for i+mIdx < len(tok.String) && isHex(tok.String[i+mIdx]) && mIdx < 2 {
					mIdx++
				}
				if mIdx == 0 {
					continue
				}

				hex, _ := strconv.ParseUint(tok.String[i:i+mIdx], 16, 8)
				str.WriteRune(rune(hex))
				i += mIdx - 1
				continue
			default:
				str.WriteByte(tok.String[i])
			}

		case '$':
			mIdx = getVarEndIndex(tok.String[i:])
			lookup, err := interp.GetVar(parseVarName(tok.String[i : i+mIdx]))
			if err != nil {
				return EmptyToken, fmt.Errorf("could not lookup var %s: %w", parseVarName(tok.String[i:mIdx]), err)
			}
			str.WriteString(lookup.String)
			i += mIdx - 1
		case '[':
			// empty subcommand [] short path
			if tok.String[i+1] == ']' {
				i += 1
				continue
			}
			mIdx := parser.FindMate(tok.String[i:], '[', ']')
			if mIdx == -1 {
				return EmptyToken, fmt.Errorf("could not find matching ] in %s", tok.Summary())
			}
			//fmt.Println("subcommand", tok.String[i+1:i+mIdx])
			ret, err := interp.ExecString(tok.String[i+1 : i+mIdx])
			if err != nil {
				return EmptyToken, fmt.Errorf("error executing subcommand %s: %w", tok.Summary(), err)
			}
			str.WriteString(ret.String)
			i += mIdx
		default:
			str.WriteByte(tok.String[i])
		}
	}

	return &Token{String: str.String()}, nil
}

// getVarEndIndex returns the index of the first char after the end of a variable name.
// Thus, given string str with value of "$asdf stuff", the return value (idx) can be used such that
// str[:idx] = "$asdf"
// The initial character in str must be the sigil ($). This is not actually checked, it is
// assumed whatever is calling getVarEndIndex as already done the check.
func getVarEndIndex(str string) (idx int) {
	if len(str) == 2 {
		return 2
	}

	if str[1] == '{' {
		// second char is a {, var name is ends at the matching }
		return parser.FindMate(str[1:], '{', '}') + 2
	}

	// otherwise, var name ends at first non-name char

	// TODO: whitelist instead of blacklist??
	idx = strings.IndexAny(str[1:], "[\\ $\n\t") + 1
	if idx == 0 {
		return len(str)
	}

	return
}

// parseVarName strips the sigil from a variable and trims off the quoting braces
// this implementation is a bit sloppy because a varname of '${{asdf}}} will
// resolve to asdf but all the parsing that happens up to this point will help.
// ${{asdf}} should technically resolve to {asdf} but this implementation will
// have it resolve to asdf.
func parseVarName(varname string) (parsed string) {
	return strings.Trim(varname[1:], "{}")
}
