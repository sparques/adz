//go:build tinygo

package repl

import "errors"

type Filer interface {
	Fd() uintptr
}

type RawMode struct{}

func EnableRawMode(_ Filer) (*RawMode, error) {
	return nil, errors.New("raw mode is not implemented on this platform")
}

func (m *RawMode) Restore() error {
	return nil
}

func RestoreTerminal() error {
	return nil
}
