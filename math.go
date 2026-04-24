package adz

import (
	"fmt"
	"strconv"
	"strings"
)

func init() {
	StdLib["eq"] = ProcEq
	StdLib["=="] = ProcEq
	StdLib["ne"] = ProcNeq
	StdLib["!="] = ProcNeq
	StdLib["not"] = ProcNot
	StdLib["and"] = ProcAnd
	StdLib["or"] = ProcOr
	StdLib["sum"] = ProcSum
	StdLib["+"] = ProcSum
	StdLib["-"] = ProcDiff
	StdLib["*"] = ProcMul
	StdLib["/"] = ProcDiv
	StdLib["incr"] = ProcIncr
	StdLib["bitand"] = ProcBitAnd
	StdLib["&"] = ProcBitAnd
	StdLib["bitor"] = ProcBitOr
	StdLib["|"] = ProcBitOr
	StdLib["bitxor"] = ProcBitXor
	StdLib["^"] = ProcBitXor
	StdLib["bitnot"] = ProcBitNot
	StdLib["bitclear"] = ProcBitClear
	StdLib["&^"] = ProcBitClear
	StdLib["lshift"] = ProcLeftShift
	StdLib["<<"] = ProcLeftShift
	StdLib["rshift"] = ProcRightShift
	StdLib[">>"] = ProcRightShift
	StdLib["lt"] = procNumericCmp(lessThan)
	StdLib["<"] = procNumericCmp(lessThan)
	StdLib["lte"] = procNumericCmp(lessThanOrEqual)
	StdLib["<="] = procNumericCmp(lessThanOrEqual)
	StdLib["gt"] = procNumericCmp(greaterThan)
	StdLib[">"] = procNumericCmp(greaterThan)
	StdLib["gte"] = procNumericCmp(greaterThanOrEqual)
	StdLib[">="] = procNumericCmp(greaterThanOrEqual)
}

type numericValue struct {
	i       int
	f       float64
	isFloat bool
}

func numericFromToken(tok *Token) (numericValue, error) {
	switch v := tok.Data.(type) {
	case int:
		return numericValue{i: v, f: float64(v)}, nil
	case Integer:
		i := v.Int()
		return numericValue{i: i, f: float64(i)}, nil
	case float64:
		return numericValue{f: v, isFloat: true}, nil
	case Floater:
		return numericValue{f: v.Float(), isFloat: true}, nil
	}

	if strings.ContainsAny(tok.String, ".eE") {
		f, err := tok.AsFloat()
		if err != nil {
			return numericValue{}, fmt.Errorf("expected number, got %q", tok.String)
		}
		return numericValue{f: f, isFloat: true}, nil
	}

	if i, err := tok.AsInt(); err == nil {
		return numericValue{i: i, f: float64(i)}, nil
	}
	f, err := tok.AsFloat()
	if err != nil {
		return numericValue{}, fmt.Errorf("expected number, got %q", tok.String)
	}
	return numericValue{f: f, isFloat: true}, nil
}

func (n numericValue) Float64() float64 {
	if n.isFloat {
		return n.f
	}
	return float64(n.i)
}

func (n numericValue) Token() *Token {
	if n.isFloat {
		return NewToken(n.f)
	}
	return NewTokenInt(n.i)
}

func (n numericValue) Add(other numericValue) numericValue {
	if n.isFloat || other.isFloat {
		return numericValue{f: n.Float64() + other.Float64(), isFloat: true}
	}
	return numericValue{i: n.i + other.i, f: float64(n.i + other.i)}
}

func (n numericValue) Sub(other numericValue) numericValue {
	if n.isFloat || other.isFloat {
		return numericValue{f: n.Float64() - other.Float64(), isFloat: true}
	}
	return numericValue{i: n.i - other.i, f: float64(n.i - other.i)}
}

func (n numericValue) Mul(other numericValue) numericValue {
	if n.isFloat || other.isFloat {
		return numericValue{f: n.Float64() * other.Float64(), isFloat: true}
	}
	return numericValue{i: n.i * other.i, f: float64(n.i * other.i)}
}

func (n numericValue) Div(other numericValue) numericValue {
	return numericValue{f: n.Float64() / other.Float64(), isFloat: true}
}

func procNumericFold(minArgs int, op func(numericValue, numericValue) numericValue) Proc {
	return func(interp *Interp, args []*Token) (*Token, error) {
		if len(args) < minArgs+1 {
			return EmptyToken, ErrArgMinimum(minArgs, len(args)-1)
		}

		acc, err := numericFromToken(args[1])
		if err != nil {
			return EmptyToken, err
		}

		for i := 2; i < len(args); i++ {
			next, err := numericFromToken(args[i])
			if err != nil {
				return EmptyToken, err
			}
			acc = op(acc, next)
		}

		return acc.Token(), nil
	}
}

func procNumericCmp(fn func(numericValue, numericValue) bool) Proc {
	return func(interp *Interp, args []*Token) (*Token, error) {
		if len(args) != 3 {
			return EmptyToken, ErrArgCount(2, len(args)-1)
		}

		a, err := numericFromToken(args[1])
		if err != nil {
			return EmptyToken, err
		}
		b, err := numericFromToken(args[2])
		if err != nil {
			return EmptyToken, err
		}

		return NewToken(fn(a, b)), nil
	}
}

func strictIntFromToken(tok *Token) (int, error) {
	switch v := tok.Data.(type) {
	case int:
		return v, nil
	case Integer:
		return v.Int(), nil
	case float64:
		return 0, fmt.Errorf("expected integer, got %q", tok.String)
	case Floater:
		return 0, fmt.Errorf("expected integer, got %q", tok.String)
	}

	i, err := strconv.Atoi(tok.String)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", tok.String)
	}
	tok.Data = i
	return i, nil
}

func procIntegerFold(minArgs int, op func(int, int) int) Proc {
	return func(interp *Interp, args []*Token) (*Token, error) {
		if len(args) < minArgs+1 {
			return EmptyToken, ErrArgMinimum(minArgs, len(args)-1)
		}

		acc, err := strictIntFromToken(args[1])
		if err != nil {
			return EmptyToken, err
		}

		for i := 2; i < len(args); i++ {
			next, err := strictIntFromToken(args[i])
			if err != nil {
				return EmptyToken, err
			}
			acc = op(acc, next)
		}

		return NewTokenInt(acc), nil
	}
}

// ProcEq performs a shallow comparison
func ProcEq(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 2; i < len(args); i++ {
		if args[1].String != args[2].String {
			return FalseToken, nil
		}
	}

	return TrueToken, nil
}

// ProcNeq performs a shallow comparison
func ProcNeq(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(args[0], 1, len(args)-1)
	}

	if args[1].String == args[2].String {
		return TrueToken, nil
	}

	return FalseToken, nil
}

// ProcNot performs a boolean not
func ProcNot(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(args[0], 1, len(args)-1)
	}

	b, err := args[1].AsBool()
	if err != nil {
		return EmptyToken, ErrExpectedBool(args[1].String)
	}

	if b {
		return FalseToken, nil
	}

	return TrueToken, nil
}

func ProcAnd(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 1; i < len(args); i++ {
		v, err := args[i].AsBool()
		if err != nil {
			return EmptyToken, ErrExpectedBool(args[0], i-1, args[i])
		}
		if !v {
			return FalseToken, nil
		}
	}

	return TrueToken, nil
}

func ProcOr(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 3 {
		return EmptyToken, ErrArgMinimum(2, len(args)-1)
	}

	for i := 1; i < len(args); i++ {
		v, err := args[i].AsBool()
		if err != nil {
			return EmptyToken, ErrExpectedBool(args[0], i-1, args[i])
		}
		if v {
			return TrueToken, nil
		}
	}

	return FalseToken, nil
}

// ProcSum
func ProcSum(interp *Interp, args []*Token) (*Token, error) {
	return procNumericFold(2, numericValue.Add)(interp, args)
}

// ProcDiff
func ProcDiff(interp *Interp, args []*Token) (*Token, error) {
	return procNumericFold(2, numericValue.Sub)(interp, args)
}

// ProcMul
func ProcMul(interp *Interp, args []*Token) (*Token, error) {
	return procNumericFold(2, numericValue.Mul)(interp, args)
}

// ProcDiv
func ProcDiv(interp *Interp, args []*Token) (*Token, error) {
	return procNumericFold(2, numericValue.Div)(interp, args)
}

// ProcIncr
func ProcIncr(interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgHelp("varName", "name of variable to increment"),
		&Argument{
			Name:    "amt",
			Default: NewToken(1),
			Help:    "amount to increase varName",
		},
	)
	as.Help = "incr increments the variable with name varName by amt"
	bound, err := as.BindPosOnly(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}
	iVar, err := interp.GetVar(bound["varName"].String)
	if err != nil {
		return EmptyToken, err // wrap for more context?
	}

	val, err := numericFromToken(iVar)
	if err != nil {
		return EmptyToken, fmt.Errorf("var %s with value of %q is not numeric", bound["varName"].String, iVar.String)
	}

	amt, err := numericFromToken(bound["amt"])
	if err != nil {
		return EmptyToken, err
	}

	interp.SetVar(bound["varName"].String, val.Add(amt).Token())

	return interp.GetVar(bound["varName"].String)
}

func ProcBitAnd(interp *Interp, args []*Token) (*Token, error) {
	return procIntegerFold(2, func(a, b int) int { return a & b })(interp, args)
}

func ProcBitOr(interp *Interp, args []*Token) (*Token, error) {
	return procIntegerFold(2, func(a, b int) int { return a | b })(interp, args)
}

func ProcBitXor(interp *Interp, args []*Token) (*Token, error) {
	return procIntegerFold(2, func(a, b int) int { return a ^ b })(interp, args)
}

func ProcBitNot(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 2 {
		return EmptyToken, ErrArgCount(1, len(args)-1)
	}

	val, err := strictIntFromToken(args[1])
	if err != nil {
		return EmptyToken, err
	}
	return NewTokenInt(^val), nil
}

func ProcBitClear(interp *Interp, args []*Token) (*Token, error) {
	return procIntegerFold(2, func(a, b int) int { return a &^ b })(interp, args)
}

func ProcLeftShift(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	lhs, err := strictIntFromToken(args[1])
	if err != nil {
		return EmptyToken, err
	}
	rhs, err := strictIntFromToken(args[2])
	if err != nil {
		return EmptyToken, err
	}
	if rhs < 0 {
		return EmptyToken, fmt.Errorf("shift count must be non-negative")
	}

	return NewTokenInt(lhs << uint(rhs)), nil
}

func ProcRightShift(interp *Interp, args []*Token) (*Token, error) {
	if len(args) != 3 {
		return EmptyToken, ErrArgCount(2, len(args)-1)
	}

	lhs, err := strictIntFromToken(args[1])
	if err != nil {
		return EmptyToken, err
	}
	rhs, err := strictIntFromToken(args[2])
	if err != nil {
		return EmptyToken, err
	}
	if rhs < 0 {
		return EmptyToken, fmt.Errorf("shift count must be non-negative")
	}

	return NewTokenInt(lhs >> uint(rhs)), nil
}

func lessThan(a, b numericValue) bool {
	if a.isFloat || b.isFloat {
		return a.Float64() < b.Float64()
	}
	return a.i < b.i
}

func lessThanOrEqual(a, b numericValue) bool {
	if a.isFloat || b.isFloat {
		return a.Float64() <= b.Float64()
	}
	return a.i <= b.i
}

func greaterThan(a, b numericValue) bool {
	if a.isFloat || b.isFloat {
		return a.Float64() > b.Float64()
	}
	return a.i > b.i
}

func greaterThanOrEqual(a, b numericValue) bool {
	if a.isFloat || b.isFloat {
		return a.Float64() >= b.Float64()
	}
	return a.i >= b.i
}
