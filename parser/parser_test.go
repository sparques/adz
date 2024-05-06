package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func Test_lineSpit(t *testing.T) {
	buf := bytes.NewBufferString("this is a line\nthis is another line\nthis line { spans multiple \n lines but it's still one 'line'}\nthis ALSO spans [multiple lines, but it uses [several] nestings] of brackets\nThis line \"uses \ndouble quotes!\"")

	scanner := bufio.NewScanner(buf)
	scanner.Split(LineSplit)

	i := 0
	for scanner.Scan() {
		fmt.Println(i, scanner.Text())
		i++
	}
}

func Test_lineSpit2(t *testing.T) {
	linescanner := bufio.NewScanner(os.Stdin)
	linescanner.Split(LineSplit)

	i := 0
	for linescanner.Scan() {
		buf := bytes.NewBufferString(linescanner.Text())
		tokenScanner := bufio.NewScanner(buf)
		tokenScanner.Split(TokenSplit)
		fmt.Print(i, " ")
		for tokenScanner.Scan() {
			fmt.Printf("%s-", tokenScanner.Text())
		}
		fmt.Println()
		i++
	}
}

func Test_FindMate(t *testing.T) {
	str := ` {asdf asdf}`
	fmt.Println("str has mate at", FindMate(str, '{', '}'), "and length of", len(str))
}

func Test_FindPair(t *testing.T) {
	str := ` "asdf \"asdf"`
	// find pair needs to find the SECOND matching quote that isn't escaped
	if str[FindPair(str, '"')] != '"' && FindPair(str, '"') != strings.IndexAny(str, `"`) {
		t.Errorf("FindPair did not return as expected.")
	}

}

func Test_CompositeTokenSplit(t *testing.T) {
	// buf := bytes.NewBufferString(`thisispartone$thisisavar`)
}
