package repl

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

const (
	keyCtrlA     = 0x01
	keyCtrlC     = 0x03
	keyCtrlD     = 0x04
	keyCtrlE     = 0x05
	keyCtrlK     = 0x0b
	keyCtrlU     = 0x15
	keyBackspace = 0x7f
	keyEnter     = '\r'
	keyLineFeed  = '\n'
	keyEscape    = 0x1b
)

var ErrInterrupted = errors.New("repl interrupted")

// LineEditor reads interactively edited lines from an input stream.
//
// It is intentionally independent from Reader: callers can use it directly for
// line editing or pass an EditorReader to NewReader for command completion.
type LineEditor struct {
	in      *bufio.Reader
	out     io.Writer
	history []string
}

// NewLineEditor returns a readline-like editor over in and out.
func NewLineEditor(in io.Reader, out io.Writer) *LineEditor {
	return &LineEditor{
		in:  bufio.NewReader(in),
		out: out,
	}
}

// ReadLine reads one edited line. Enter submits the line. Ctrl-Enter inserts a
// newline into the buffer when the terminal reports a distinguishable
// Ctrl-Enter sequence.
func (e *LineEditor) ReadLine(prompt string) (string, error) {
	if prompt != "" {
		if _, err := io.WriteString(e.out, prompt); err != nil {
			return "", err
		}
	}

	var buf []rune
	cursor := 0
	historyPos := len(e.history)
	draft := ""
	for {
		r, _, err := e.in.ReadRune()
		if err != nil {
			if err == io.EOF && len(buf) > 0 {
				return string(buf), nil
			}
			return "", err
		}

		switch r {
		case keyCtrlA:
			e.moveLeft(cursor)
			cursor = 0
		case keyCtrlC:
			io.WriteString(e.out, "^C\n")
			return "", ErrInterrupted
		case keyCtrlD:
			if len(buf) == 0 {
				return "", io.EOF
			}
			if cursor < len(buf) {
				buf = append(buf[:cursor], buf[cursor+1:]...)
				e.redrawTail(buf, cursor)
			}
		case keyCtrlE:
			e.moveRight(len(buf) - cursor)
			cursor = len(buf)
		case keyCtrlK:
			if cursor < len(buf) {
				clear := len(buf) - cursor
				buf = buf[:cursor]
				e.clearRunes(clear)
				e.moveLeft(clear)
			}
		case keyCtrlU:
			if cursor > 0 {
				oldLen := len(buf)
				buf = append([]rune(nil), buf[cursor:]...)
				e.moveLeft(cursor)
				io.WriteString(e.out, string(buf))
				clear := oldLen - len(buf)
				e.clearRunes(clear)
				e.moveLeft(len(buf) + clear)
				cursor = 0
			}
		case keyEnter, keyLineFeed:
			io.WriteString(e.out, "\r\n")
			line := string(buf)
			e.addHistory(line)
			return line, nil
		case keyBackspace, '\b':
			if cursor > 0 {
				cursor--
				buf = append(buf[:cursor], buf[cursor+1:]...)
				io.WriteString(e.out, "\b")
				e.redrawTail(buf, cursor)
			}
		case keyEscape:
			action, err := e.readEscape()
			if err != nil {
				return "", err
			}
			switch action {
			case editInsertNewline:
				buf = append(buf, '\n')
				cursor = len(buf)
				io.WriteString(e.out, "\r\n")
			case editLeft:
				if cursor > 0 {
					cursor--
					e.moveLeft(1)
				}
			case editRight:
				if cursor < len(buf) {
					cursor++
					e.moveRight(1)
				}
			case editHome:
				e.moveLeft(cursor)
				cursor = 0
			case editEnd:
				e.moveRight(len(buf) - cursor)
				cursor = len(buf)
			case editHistoryPrev:
				if historyPos > 0 {
					if historyPos == len(e.history) {
						draft = string(buf)
					}
					historyPos--
					buf, cursor = e.replaceLine(buf, cursor, e.history[historyPos])
				}
			case editHistoryNext:
				if historyPos < len(e.history) {
					historyPos++
					if historyPos == len(e.history) {
						buf, cursor = e.replaceLine(buf, cursor, draft)
					} else {
						buf, cursor = e.replaceLine(buf, cursor, e.history[historyPos])
					}
				}
			case editDelete:
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
					e.redrawTail(buf, cursor)
				}
			}
		default:
			buf = append(buf, 0)
			copy(buf[cursor+1:], buf[cursor:])
			buf[cursor] = r
			suffix := string(buf[cursor+1:])
			if _, err := io.WriteString(e.out, string(r)+suffix); err != nil {
				return "", err
			}
			cursor++
			if len(suffix) > 0 {
				e.moveLeft(len([]rune(suffix)))
			}
		}
	}
}

type editAction int

const (
	editNoop editAction = iota
	editInsertNewline
	editLeft
	editRight
	editHome
	editEnd
	editHistoryPrev
	editHistoryNext
	editDelete
)

func (e *LineEditor) readEscape() (editAction, error) {
	seq := []byte{keyEscape}
	for len(seq) < 16 {
		b, err := e.in.ReadByte()
		if err != nil {
			if err == io.EOF {
				return editNoop, nil
			}
			return editNoop, err
		}
		seq = append(seq, b)
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
			break
		}
	}

	// Common encodings for Ctrl-Enter:
	//   CSI u:       ESC [ 13;5u
	//   modifyOther: ESC [ 27;5;13~
	if bytes.Equal(seq, []byte{keyEscape, '[', '1', '3', ';', '5', 'u'}) ||
		bytes.Equal(seq, []byte{keyEscape, '[', '2', '7', ';', '5', ';', '1', '3', '~'}) {
		return editInsertNewline, nil
	}

	switch string(seq) {
	case "\x1b[D", "\x1bb":
		return editLeft, nil
	case "\x1b[C", "\x1bf":
		return editRight, nil
	case "\x1b[A", "\x1bOA":
		return editHistoryPrev, nil
	case "\x1b[B", "\x1bOB":
		return editHistoryNext, nil
	case "\x1b[H", "\x1b[1~", "\x1bOH":
		return editHome, nil
	case "\x1b[F", "\x1b[4~", "\x1bOF":
		return editEnd, nil
	case "\x1b[3~":
		return editDelete, nil
	}

	return editNoop, nil
}

func (e *LineEditor) addHistory(line string) {
	if line == "" {
		return
	}
	if len(e.history) > 0 && e.history[len(e.history)-1] == line {
		return
	}
	e.history = append(e.history, line)
	if len(e.history) > 16 {
		copy(e.history, e.history[len(e.history)-16:])
		e.history = e.history[:16]
	}
}

func (e *LineEditor) replaceLine(buf []rune, cursor int, line string) ([]rune, int) {
	e.moveLeft(cursor)
	e.clearRunes(len(buf))
	e.moveLeft(len(buf))
	newBuf := []rune(line)
	io.WriteString(e.out, line)
	return newBuf, len(newBuf)
}

func (e *LineEditor) redrawTail(buf []rune, cursor int) {
	tail := string(buf[cursor:])
	io.WriteString(e.out, tail+" ")
	e.moveLeft(len([]rune(tail)) + 1)
}

func (e *LineEditor) moveLeft(n int) {
	for ; n > 0; n-- {
		io.WriteString(e.out, "\x1b[D")
	}
}

func (e *LineEditor) moveRight(n int) {
	for ; n > 0; n-- {
		io.WriteString(e.out, "\x1b[C")
	}
}

func (e *LineEditor) clearRunes(n int) {
	for ; n > 0; n-- {
		io.WriteString(e.out, " ")
	}
}

// PromptFunc returns the prompt for the next line. continued is true after at
// least one edited line has been submitted for the current command.
type PromptFunc func(continued bool) string

// EditorReader adapts a LineEditor to io.Reader so it can be composed with
// Reader. It appends a newline after each submitted edited line.
type EditorReader struct {
	editor *LineEditor
	prompt PromptFunc
	buf    bytes.Buffer
	seen   bool
}

// NewEditorReader returns an io.Reader backed by editor.
func NewEditorReader(editor *LineEditor, prompt PromptFunc) *EditorReader {
	return &EditorReader{
		editor: editor,
		prompt: prompt,
	}
}

func (r *EditorReader) Read(p []byte) (int, error) {
	if r.buf.Len() == 0 {
		prompt := ""
		if r.prompt != nil {
			prompt = r.prompt(r.seen)
		}
		line, err := r.editor.ReadLine(prompt)
		if err != nil {
			return 0, err
		}
		r.seen = true
		r.buf.WriteString(line)
		r.buf.WriteByte('\n')
	}

	return r.buf.Read(p)
}

// ResetCommand tells the reader that the previous command has completed, so
// the next line should use the primary prompt.
func (r *EditorReader) ResetCommand() {
	r.seen = false
}
