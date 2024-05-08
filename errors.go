package adz

import (
	"errors"
	"fmt"
)

var (
	ErrReturn   = errors.New("return")
	ErrBreak    = errors.New("break")
	ErrContinue = errors.New("continue")
	ErrTailcall = errors.New("tailcall")
)

var (
	ErrSyntax               = Error(errSyntax)
	ErrSyntaxExpected       = Error(errSyntaxExpected)
	ErrEvalCond             = Error(errEvalCond)
	ErrCondNotBool          = Error(errCondNotBool)
	ErrNoVar                = Error(errNoVar)
	ErrArgCount             = Error(errArgCount)
	ErrArgMinimum           = Error(errArgMinimum)
	ErrExpectedBool         = Error(errExpectedBool)
	ErrExpectedInt          = Error(errExpectedInt)
	ErrNamedArgMissingValue = Error(errNamedArgMissingValue)
)

type Error func(...any) error

func (e Error) Error() string {
	return e().Error()
}

func errSyntax(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("syntax error: %v", args[0])
	case 2:
		return fmt.Errorf("syntax error: %v: %s", args[0], args[1])
	default:
		return fmt.Errorf("syntax error")
	}
}

func errSyntaxExpected(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("expected %s, got %s", args[0], args[1])
	case 3:
		return fmt.Errorf("%s: expected %s, got %s", args[0], args[1], args[2])
	default:
		return fmt.Errorf("syntax not expected")
	}
}

func errEvalCond(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("error evaluating condition: %v", args[0])
	case 2:
		return fmt.Errorf("error evaluating %v condition: %v", args[0], args[1])
	default:
		return fmt.Errorf("error evaluating condition")
	}
}

func errCondNotBool(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("condition returned a non-bool value: %v", args[0])
	case 2:
		return fmt.Errorf("condition for %v returned a non-bool value: %v", args[0], args[1])
	default:
		return fmt.Errorf("condition returned a non-bool value")
	}
}

func errNoVar(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("no such variable %v", args[0])
	default:
		return fmt.Errorf("no such variable")
	}
}

func errArgCount(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("expected %d args, got %d", args[0], args[1])
	case 3:
		return fmt.Errorf("%v: expected %v args, got %v", args[0], args[1], args[2])
	default:
		return fmt.Errorf("wrong number of args")
	}
}

func errArgMinimum(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("expected at least %d args, got %d", args[0], args[1])
	default:
		return fmt.Errorf("minimum args not met")
	}
}

func errExpectedBool(args ...any) error {
	switch len(args) {
	case 4:
		return fmt.Errorf("%v: could not parse arg #%v %v as bool: %v", args[0], args[1], args[2], args[3])
	case 3:
		return fmt.Errorf("%v: could not parse arg #%v as bool: %v", args[0], args[1], args[2])
	default:
		return fmt.Errorf("expected bool")
	}
}

func errExpectedInt(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("expected integer, got %v", args[0])
	default:
		return fmt.Errorf("expected integer")
	}
}

func errNamedArgMissingValue(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("named arg %v missing value", args[0])
	default:
		return fmt.Errorf("named arg missing value")
	}
}

type Error2 struct {
	offset int
	Err    error
}
