package repl

import (
	"bufio"
	"io"

	"github.com/sparques/adz"
	"github.com/sparques/adz/parser"
)

// InteractiveReader reads complete commands from interactively edited lines.
// It uses the same parser.LineSplit completion rule as Reader.
type InteractiveReader struct {
	editor  *LineEditor
	prompt  PromptFunc
	pending []byte
	cfg     readerConfig
}

// NewInteractiveReader returns a command reader backed by an interactive line
// editor.
func NewInteractiveReader(editor *LineEditor, prompt PromptFunc, opts ...ReaderOption) *InteractiveReader {
	cfg := readerConfig{
		maxCommandSize: defaultMaxCommandSize,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &InteractiveReader{
		editor: editor,
		prompt: prompt,
		cfg:    cfg,
	}
}

// ReadCommand reads the next complete command.
func (r *InteractiveReader) ReadCommand() (*Command, error) {
	for {
		cmd, ok, err := r.nextPendingCommand(false)
		if ok || err != nil {
			return cmd, err
		}

		prompt := ""
		if r.prompt != nil {
			prompt = r.prompt(len(r.pending) > 0)
		}

		line, err := r.editor.ReadLine(prompt)
		if err != nil {
			if err == io.EOF && len(r.pending) > 0 {
				return r.nextPendingEOF()
			}
			r.pending = nil
			return nil, err
		}

		r.pending = append(r.pending, line...)
		r.pending = append(r.pending, '\n')
		if len(r.pending) > r.cfg.maxCommandSize {
			r.pending = nil
			return nil, bufio.ErrTooLong
		}
	}
}

func (r *InteractiveReader) nextPendingEOF() (*Command, error) {
	cmd, ok, err := r.nextPendingCommand(true)
	if ok || err != nil {
		return cmd, err
	}
	return nil, io.EOF
}

func (r *InteractiveReader) nextPendingCommand(atEOF bool) (*Command, bool, error) {
	if len(r.pending) == 0 {
		return nil, false, nil
	}

	advance, token, err := parser.LineSplit(r.pending, atEOF)
	if err != nil {
		return nil, false, err
	}
	if token == nil {
		return nil, false, nil
	}

	text := append([]byte(nil), token...)
	if advance == len(r.pending) {
		r.pending = r.pending[:0]
	} else {
		r.pending = r.pending[:copy(r.pending, r.pending[advance:])]
	}
	script, err := adz.LexBytes(text)
	if err != nil {
		return nil, false, err
	}

	return &Command{
		Text:   text,
		Script: script,
	}, true, nil
}
