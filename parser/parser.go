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

func FindMateByte(s []byte, openSymbol, closeSymbol byte) int {
	var count int
	seenOpen := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 < len(s) {
				i++
			}
		case openSymbol:
			count++
			seenOpen = true
		case closeSymbol:
			if seenOpen {
				count--
				if count == 0 {
					return i
				}
			}
		}
	}
	return -1
}

// LineSplit is a bufio.Scanner SplitFunc. It splits a stream into "lines" but honors escapes
// and quoting brackets / braces.
func LineSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	count := 0
	var symbolIncr, symbolDecr byte

	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\\':
			if i+1 < len(data) {
				i++
			}
		case '\n', ';':
			if count == 0 {
				return i + 1, dropCR(data[0:i]), nil
			}
		case '"':
			if count == 0 {
				symbolIncr = '"'
				symbolDecr = '"'
				count = 1
			} else if symbolIncr == '"' {
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
			} else if data[i] == symbolIncr {
				count++
			}
		}
	}
	if atEOF {
		return len(data), dropCR(data), nil
	}
	return 0, nil, nil
}

func TokenSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Skip leading space
	start := 0
	for ; start < len(data) && isSpace(data[start]); start++ {
	}
	if start >= len(data) {
		return len(data), nil, nil
	}

	count := 0
	var symbolIncr, symbolDecr byte

	for i := start; i < len(data); i++ {
		switch data[i] {
		case '\\':
			if i+1 < len(data) {
				i++
			}
		case ' ', '\t', '\n', '\r', '\f':
			if count == 0 {
				return i + 1, dropCR(data[start:i]), nil
			}
		case '"':
			if count == 0 {
				symbolIncr = '"'
				symbolDecr = '"'
				count = 1
			} else if symbolIncr == '"' {
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
			} else if data[i] == symbolIncr {
				count++
			}
		}
	}
	if atEOF {
		return len(data), dropCR(data[start:]), nil
	}
	return 0, nil, nil
}
