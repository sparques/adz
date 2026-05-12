package repl

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReaderReadsCompleteCommands(t *testing.T) {
	r := NewReader(strings.NewReader("cmd1 arg1\nproc x {\nprint hi\n}\ncmd2;cmd3\n"))

	want := []string{
		"cmd1,arg1",
		"proc,x,{\nprint hi\n}",
		"cmd2",
		"cmd3",
	}

	for i, expected := range want {
		cmd, err := r.ReadCommand()
		if err != nil {
			t.Fatalf("ReadCommand %d: %v", i+1, err)
		}
		if got := strings.Join(commandStrings(cmd), "|"); got != expected {
			t.Fatalf("command %d = %q, want %q", i+1, got, expected)
		}
	}

	if _, err := r.ReadCommand(); !errors.Is(err, io.EOF) {
		t.Fatalf("final ReadCommand err = %v, want EOF", err)
	}
}

func TestLineEditorCtrlEnterInsertsNewline(t *testing.T) {
	var out strings.Builder
	editor := NewLineEditor(strings.NewReader("first\x1b[13;5usecond\n"), &out)

	line, err := editor.ReadLine("? ")
	if err != nil {
		t.Fatalf("ReadLine: %v", err)
	}
	if line != "first\nsecond" {
		t.Fatalf("line = %q", line)
	}
	if !strings.Contains(out.String(), "? first\r\nsecond\r\n") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestLineEditorEditsAtCursor(t *testing.T) {
	var out strings.Builder
	editor := NewLineEditor(strings.NewReader("ac\x1b[Db\n"), &out)

	line, err := editor.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine: %v", err)
	}
	if line != "abc" {
		t.Fatalf("line = %q", line)
	}
}

func TestInteractiveReaderRecoversFromInterrupt(t *testing.T) {
	var out strings.Builder
	editor := NewLineEditor(strings.NewReader("unfinished {\n\x03cmd\n"), &out)
	reader := NewInteractiveReader(editor, func(continued bool) string {
		if continued {
			return "> "
		}
		return "? "
	})

	if _, err := reader.ReadCommand(); !errors.Is(err, ErrInterrupted) {
		t.Fatalf("ReadCommand interrupt err = %v, want ErrInterrupted", err)
	}

	cmd, err := reader.ReadCommand()
	if err != nil {
		t.Fatalf("ReadCommand after interrupt: %v", err)
	}
	if got := strings.Join(commandStrings(cmd), "|"); got != "cmd" {
		t.Fatalf("command after interrupt = %q", got)
	}
}

func commandStrings(cmd *Command) []string {
	var out []string
	for _, command := range cmd.Script {
		var tokens []string
		for _, tok := range command {
			tokens = append(tokens, tok.String)
		}
		out = append(out, strings.Join(tokens, ","))
	}
	return out
}
