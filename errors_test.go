package adz

import (
	"errors"
	"testing"
)

func Test_ErrorMatch(t *testing.T) {
	err1 := ErrSyntax("wow, not even close")

	if !errors.Is(err1, ErrSyntax) {
		t.Errorf("errors didn't match but were supposed to")
	}
}
