package repl

import (
	"bufio"
	"io"

	"github.com/sparques/adz"
	"github.com/sparques/adz/parser"
)

const defaultMaxCommandSize = 1024 * 1024

// Command is one complete REPL command as read from the input stream.
type Command struct {
	// Text is the raw command text before lexing.
	Text []byte
	// Script is the lexed adz script for Text.
	Script adz.Script
}

// Reader reads complete adz commands from an io.Reader.
//
// Completion is determined by adz's parser.LineSplit, so newlines and
// semicolons only end a command when braces, brackets, and quotes are balanced.
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader returns a Reader that reads complete commands from r.
func NewReader(r io.Reader, opts ...ReaderOption) *Reader {
	cfg := readerConfig{
		maxCommandSize: defaultMaxCommandSize,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	scanner := bufio.NewScanner(r)
	scanner.Split(parser.LineSplit)
	scanner.Buffer(make([]byte, 0, 64*1024), cfg.maxCommandSize)

	return &Reader{scanner: scanner}
}

// ReadCommand reads the next complete command.
func (r *Reader) ReadCommand() (*Command, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	text := append([]byte(nil), r.scanner.Bytes()...)
	script, err := adz.LexBytes(text)
	if err != nil {
		return nil, err
	}

	return &Command{
		Text:   text,
		Script: script,
	}, nil
}

type readerConfig struct {
	maxCommandSize int
}

// ReaderOption configures a Reader.
type ReaderOption func(*readerConfig)

// WithMaxCommandSize sets the largest command that Reader will buffer.
func WithMaxCommandSize(n int) ReaderOption {
	return func(cfg *readerConfig) {
		if n > 0 {
			cfg.maxCommandSize = n
		}
	}
}
