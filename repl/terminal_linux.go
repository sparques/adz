//go:build linux && !tinygo

package repl

import (
	"sync"
	"syscall"
	"unsafe"
)

// Filer is implemented by readers that expose an operating-system file
// descriptor. It is intentionally smaller than *os.File so callers can keep
// their APIs expressed in terms of io.Reader.
type Filer interface {
	Fd() uintptr
}

// RawMode restores a terminal after raw mode is disabled.
type RawMode struct {
	fd     uintptr
	old    syscall.Termios
	active bool
}

var rawModes = struct {
	sync.Mutex
	active map[uintptr]*RawMode
}{
	active: make(map[uintptr]*RawMode),
}

// EnableRawMode puts f into a small raw mode suitable for LineEditor.
func EnableRawMode(f Filer) (*RawMode, error) {
	fd := f.Fd()

	var old syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, uintptr(unsafe.Pointer(&old))); err != nil {
		return nil, err
	}

	raw := old
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := ioctl(fd, syscall.TCSETS, uintptr(unsafe.Pointer(&raw))); err != nil {
		return nil, err
	}

	mode := &RawMode{fd: fd, old: old, active: true}
	rawModes.Lock()
	rawModes.active[fd] = mode
	rawModes.Unlock()

	return mode, nil
}

// Restore disables raw mode.
func (m *RawMode) Restore() error {
	if m == nil {
		return nil
	}
	rawModes.Lock()
	defer rawModes.Unlock()
	return m.restoreLocked()
}

func (m *RawMode) restoreLocked() error {
	if !m.active {
		return nil
	}
	m.active = false
	if rawModes.active[m.fd] == m {
		delete(rawModes.active, m.fd)
	}
	return ioctl(m.fd, syscall.TCSETS, uintptr(unsafe.Pointer(&m.old)))
}

// RestoreTerminal restores any terminal state modified by EnableRawMode.
func RestoreTerminal() error {
	rawModes.Lock()
	defer rawModes.Unlock()

	var firstErr error
	for _, mode := range rawModes.active {
		if err := mode.restoreLocked(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func ioctl(fd uintptr, request uint, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(request), arg)
	if errno != 0 {
		return errno
	}
	return nil
}
