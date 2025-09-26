package adz

import (
	"fmt"
)

var (
	ErrFlowControl          = flowControl("")
	ErrReturn               = flowControl("return")
	ErrBreak                = flowControl("break")
	ErrContinue             = flowControl("continue")
	ErrTailcall             = flowControl("tailcall")
	ErrMaxCallDepthExceeded = flowControl("max call depth exceeded")
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

type UsageError struct {
	msg string
}

func (ue *UsageError) Error() string {
	return ue.msg
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
		return adzError("error")
	}
}

func errExpectedMore(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("expected more after %v", args[0])
	case 2:
		return fmt.Errorf("expected %v after %v", args[0], args[1])
	default:
		return adzError("expected more tokens")
	}
}

func errSyntaxExpected(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("expected %s, got %s", args[0], args[1])
	case 3:
		return fmt.Errorf("%s: expected %s, got %s", args[0], args[1], args[2])
	default:
		return adzError("syntax not expected")
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
		return adzError("error evaluating conditional expression")
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
		return adzError("error evaluating body")
	}
}

func errCondNotBool(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %q", errCondNotBool(), args[0])
	case 2:
		return fmt.Errorf("%w (%v): %q", errCondNotBool(), args[0], args[1])
	default:
		return adzError("condition returned a non-bool value")
	}
}

func errNoVar(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w %q", errNoVar(), args[0])
	default:
		return adzError("no such variable")
	}
}

func errNoNamespace(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w %q", errNoNamespace(), args[0])
	default:
		return adzError("no such namespace")
	}
}

func errArgCount(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("%w: expected %v positional args, got %d", errArgCount(), args[0], args[1])
	case 3:
		return fmt.Errorf("%w: %v: expected %v positional args, got %v", errArgCount(), args[0], args[1], args[2])
	default:
		return adzError("wrong number of args")
	}
}

func errArgMinimum(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("%w, expected > %d args, got %d", errArgMinimum(), args[0], args[1])
	default:
		return adzError("minimum args not met")
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
		return fmt.Errorf("%w: expected %q, got %q: ", errExpectedArgType(), args[1], args[0])
	default:
		return adzError("arg is not expected type")
	}
}

func errExpectedBool(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w, got %v", errExpectedBool(), args[0])
	default:
		return adzError("expected bool")
	}
}

func errExpectedInt(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w, got %v", errExpectedInt(), args[0])
	default:
		return adzError("expected integer")
	}
}

func errExpectedList(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %v", errExpectedList(), args[0])
	default:
		return adzError("expected a list")
	}
}

func errNamedArgMissingValue(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %v", errNamedArgMissingValue(), args[0])
	default:
		return adzError("named arg missing value")
	}
}

func stackWrap(cmd string, arg, line int, err error) error {
	return fmt.Errorf("%s, arg %d, line %d: %w", cmd, arg, line, err)
}

func errCommand(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("%w%v: %v", errCommand(), args[0], args[1])
	default:
		return adzError("")
	}
}

func errLine(args ...any) error {
	switch len(args) {
	case 2:
		return fmt.Errorf("%w %v: %v", errLine(), args[0], args[1])
	default:
		return adzError("line")
	}
}

func errNotImplemented(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %v", errNotImplemented(), args[0])
	default:
		return adzError("not implemented")
	}
}

func errMaxCallDepthExceeded(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w: %v", errMaxCallDepthExceeded(), args[0])
	default:
		return adzError("max call depth exceeded")
	}
}

func errGoPanic(args ...any) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("%w while executing command: %v", errGoPanic(), args[0])
	default:
		return adzError("go panic")
	}
}
