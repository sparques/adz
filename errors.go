package adz

import (
	"fmt"
)

var (
	ErrFlowControl = flowControl("")
	ErrReturn      = flowControl("return")
	ErrBreak       = flowControl("break")
	ErrContinue    = flowControl("continue")
	ErrTailcall    = flowControl("tailcall")
)

var (
	ErrCommandNotFound      = Error(errCommandNotFound)
	ErrSyntax               = Error(errSyntax)
	ErrExpectedMore         = Error(errExpectedMore)
	ErrSyntaxExpected       = Error(errSyntaxExpected)
	ErrEvalCond             = Error(errEvalCond)
	ErrEvalBody             = Error(errEvalBody)
	ErrCondNotBool          = Error(errCondNotBool)
	ErrNoVar                = Error(errNoVar)
	ErrNoNamespace          = Error(errNoNamespace)
	ErrArgCount             = Error(errArgCount)
	ErrArgMinimum           = Error(errArgMinimum)
	ErrArgMissing           = Error(errArgMissing)
	ErrArgExtra             = Error(errArgExtra)
	ErrExpectedArgType      = Error(errExpectedArgType)
	ErrExpectedBool         = Error(errExpectedBool)
	ErrExpectedInt          = Error(errExpectedInt)
	ErrExpectedList         = Error(errExpectedList)
	ErrNamedArgMissingValue = Error(errNamedArgMissingValue)
	ErrCommand              = Error(errCommand)
	ErrLine                 = Error(errLine)
	ErrNotImplemented       = Error(errNotImplemented)
	ErrMaxCallDepthExceeded = Error(errMaxCallDepthExceeded)
	ErrGoPanic              = Error(errGoPanic)
)

type Error func(...any) error

func (e Error) Error() string {
	return e().Error()
}

func (e Error) Is(target error) bool {

	if target == nil {
		return false
	}
	return target.Error() == e().Error()
}

type flowControl string

func (fc flowControl) Error() string {
	return string(fc)
}

func (fc flowControl) Is(target error) bool {
	if target == ErrFlowControl {
		return true
	}
	return false
}

type adzError string

func (ae adzError) Error() string {
	return string(ae)
}

func (ae adzError) Is(target error) bool {
	return string(ae) == target.Error()
}

type Errors []error

func (e Errors) Error() string {
	return e[0].Error()
}

func (e *Errors) Append(err error) {
	*e = append(*e, err)
}

func JoinErr(a, b error) error {
	var all Errors
	if as, ok := a.(Errors); ok {
		all = append(all, as...)
	} else {
		all = append(all, a)
	}

	if bs, ok := b.(Errors); ok {
		all = append(all, bs...)
	} else {
		all = append(all, b)
	}
	return all
}

// JoinErr(ErrLine(line),ErrSyntax)

// func (ae adzError) Is(target error) bool {
// 	return string(ae) == target.Error()
// }

// type ThrownError Error
//
// func (ThrownError) Is(err, target error) bool {
// 	ErrThrow
// }

// func errThrow(args ...any) error {
//
// }

func errCommandNotFound(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %v", errCommandNotFound(), args[0])
	default:
		return adzError("command not found")
	}
}

func errSyntax(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %v", errSyntax(), args[0])
	case 2:
		return fmt.Errorf("%w: %v: %v", errSyntax(), args[0], args[1])
	default:
		return adzError("syntax error")
	}
}

func errSubst(args ...any) error {
	switch len(args) {
	default:
		return fmt.Errorf("error")
	}
}

func errExpectedMore(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("expected more after %v", args[0])
	case 2:
		return fmt.Errorf("expected %v after %v", args[0], args[1])
	default:
		return fmt.Errorf("expected more tokens")
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
		return fmt.Errorf("condition expression: %v", args[0])
	case 2:
		return fmt.Errorf("arg %v: conditional expression: %v", args[0], args[1])
	case 3:
		return fmt.Errorf("arg %v: conditional expression for %v: %v", args[0], args[1], args[2])
	default:
		return fmt.Errorf("error evaluating conditional expression")
	}
}

func errEvalBody(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("evaluating body: %v", args[0])
	case 2:
		return fmt.Errorf("evaluating %v body: %v", args[0], args[1])
	case 3:
		return fmt.Errorf("arg %v: evaluating %v body: %v", args[0], args[1], args[2])
	default:
		return fmt.Errorf("error evaluating body")
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

func errNoNamespace(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("no such namespace %v", args[0])
	default:
		return fmt.Errorf("no such namespace")
	}
}

func errArgCount(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("expected %v positional args, got %d", args[0], args[1])
	case 3:
		return fmt.Errorf("%v: expected %v positional args, got %v", args[0], args[1], args[2])
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

func errArgMissing(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w %v", errArgMissing(), args[0])
	default:
		return adzError("missing required arg")
	}
}

func errArgExtra(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w %v", errArgExtra(), args[0])
	default:
		return adzError("got extra arg")
	}
}

func errExpectedArgType(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("%s: %w, expected %s", args[0], errExpectedArgType(), args[1])
	default:
		return adzError("arg is not expected type")
	}
}

func errExpectedBool(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("expected bool, got %v", args[0])
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

func errExpectedList(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("expected a list: %v", args[0])
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

func errCommand(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("%v: %v", args[0], args[1])
	default:
		return fmt.Errorf("error evaluating command")
	}
}

func errLine(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("line %v: %v", args[0], args[1])
	default:
		return fmt.Errorf("error evaluating command")
	}
}

func errNotImplemented(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%v: not implemented", args[0])
	default:
		return fmt.Errorf("not implemented")
	}
}

func errMaxCallDepthExceeded(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("max call depth exceeded: %v", args[0])
	default:
		return fmt.Errorf("max call depth exceeded")
	}
}

func errGoPanic(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("go panic while executing command: %v", args[0])
	default:
		return fmt.Errorf("go panic")
	}
}
