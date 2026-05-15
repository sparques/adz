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

func TestLineEditorHistoryUpDown(t *testing.T) {
	var out strings.Builder
	editor := NewLineEditor(strings.NewReader("first\nsecond\n\x1b[A\ntail\x1b[A\x1b[B\n"), &out)

	line, err := editor.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine first: %v", err)
	}
	if line != "first" {
		t.Fatalf("first line = %q", line)
	}

	line, err = editor.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine second: %v", err)
	}
	if line != "second" {
		t.Fatalf("second line = %q", line)
	}

	line, err = editor.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine recalled previous: %v", err)
	}
	if line != "second" {
		t.Fatalf("previous history line = %q", line)
	}

	line, err = editor.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine recalled next: %v", err)
	}
	if line != "tail" {
		t.Fatalf("next history restored draft = %q", line)
	}
}

func TestLineEditorHistoryKeepsLast16(t *testing.T) {
	var input strings.Builder
	for i := 0; i < 17; i++ {
		input.WriteString("cmd")
		input.WriteByte(byte('a' + i))
		input.WriteByte('\n')
	}
	for i := 0; i < 16; i++ {
		input.WriteString("\x1b[A")
	}
	input.WriteByte('\n')

	var out strings.Builder
	editor := NewLineEditor(strings.NewReader(input.String()), &out)
	for i := 0; i < 17; i++ {
		line, err := editor.ReadLine("")
		if err != nil {
			t.Fatalf("ReadLine seed %d: %v", i, err)
		}
		want := "cmd" + string(rune('a'+i))
		if line != want {
			t.Fatalf("seed line %d = %q, want %q", i, line, want)
		}
	}

	line, err := editor.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine oldest retained: %v", err)
	}
	if line != "cmdb" {
		t.Fatalf("oldest retained line = %q, want %q", line, "cmdb")
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
