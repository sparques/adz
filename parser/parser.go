package parser

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func FindMate(s string, openSymbol, closeSymbol byte) int {
	var count int
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case openSymbol:
			count++
		case closeSymbol:
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

// Find first matching, unescaped reocurrence of symbol. For finding matching double quotes.
func FindPair(s string, symbol byte) int {
	var count int
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case symbol:
			count++
			if count > 1 {
				return i
			}
		}
	}
	return -1
}

func FindMateByte(s []byte, openSymbol, closeSymbol byte) int {
	var count int
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case closeSymbol:
			count--
		case openSymbol:
			count++
		}
		if count == 0 {
			return i
		}
	}
	return -1
}

func closeSymbol(b byte) byte {
	switch b {
	case '{':
		return '}'
	case '[':
		return ']'
	case '"':
		return '"'
	}
	return 0
}

// LineSplit is a bufio.Scanner SplitFunc. It splits a stream into "lines" but honors escapes
// and quoting brackets / braces.
func LineSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	count := 0
	var symbolIncr, symbolDecr byte
	//var escape bool
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\\':
			i++
		case '\n', ';':
			if count == 0 {
				return i + 1, dropCR(data[0:i]), nil
			}
		case '"':
			if count == 0 {
				symbolIncr = data[i]
				count++
				continue
			}
			if count > 0 && data[i] == symbolIncr {
				count = 0
			}
		case '}', ']':
			if count > 0 && data[i] == symbolDecr {
				count--
			}
		case '{', '[':
			if count == 0 {
				symbolIncr = data[i]
				symbolDecr = closeSymbol(data[i])
				count++
				continue
			}
			if data[i] == symbolIncr {
				count++
			}
		}
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}

	// Request more data.
	return 0, nil, nil
}

func IsName(b byte) bool {
	switch b {
	case '[', ']', '"', '{', '}', ' ', '\t', '\r', '\n', '\f':
		return false
	}
	return true
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\r', '\n', '\f':
		return true
	}

	return false
}

func TokenSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Skip leading spaces.
	start := 0
	for ; start < len(data) && isSpace(data[start]); start++ {
	}

	// we're at the end of data and it was just white space; advance by that much,
	// but don't return a token
	if start >= len(data) {
		return len(data), nil, nil
	}

	count := 0
	var symbolIncr, symbolDecr byte
	//var escape bool
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '\\':
			i++
		case ' ', '\t', '\n':
			if count == 0 {
				return i + 1, dropCR(data[start:i]), nil
			}
		case '"':
			if count == 0 {
				symbolIncr = data[i]
				count++
				continue
			}
			if count > 0 && data[i] == symbolIncr {
				count = 0
			}
		case '}', ']':
			if count > 0 && data[i] == symbolDecr {
				count--
			}
		case '{', '[':
			if count == 0 {
				symbolIncr = data[i]
				symbolDecr = closeSymbol(data[i])
				count = 1
				continue
			}
			if data[i] == symbolIncr {
				count++
			}
		}
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data[start:]), nil
	}

	// Request more data.
	return 0, nil, nil
}

/*
func CompositeTokenSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	advance = bytes.IndexAny(data, `\\[$`)
	if advance == -1 {
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	if advance != 0 {
		return advance, data[:advance], nil
	}

	switch data[0] {
	case '$':
		// eat while IsName or to matching }
	case '[':
		// eat to matching ]
	case '\\':
		// single escapes, eat next char,
		// u eats all following hex digits, and x eats 2 following hex digits
	}

}
*/
