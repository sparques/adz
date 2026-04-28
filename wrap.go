package adz

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// helper: build a tuple coercer token: ["tuple", ["M1","M2",...]]
func tupleCoercer(methods []string) *Token {
	return NewList(
		[]*Token{
			NewTokenString("tuple"),
			NewList(NewTokenListString(methods)),
		})
}

// Wrap takes any value v and returns a *Token that prints like v and
// is invocable as a Proc: [$obj <Method> arg ...]. Method is validated
// via ArgSet using a tuple coercer that lists all valid methods.
func Wrap(v any) *Token {
	recv := reflect.ValueOf(v)
	methods := collectBoundMethods(recv)

	// Build ArgSet: method (tuple-coerced) + variadic args
	names := make([]string, 0, len(methods))
	for n := range methods {
		names = append(names, n)
	}
	sort.Strings(names)

	as := NewArgSet(fmt.Sprintf("%T", v),
		&Argument{
			Name:   "method",
			Coerce: tupleCoercer(names), // validates and prints allowed methods
			Help:   "Method name",
		},
		&Argument{
			Name: "args", // marks variadic positional
			Help: "Arguments passed to the method",
		},
	)

	// no extra validation required, but ensures all flags have been set correctly
	_ = as.Validate()

	proc := Proc(func(interp *Interp, argv []*Token) (*Token, error) {
		// Bind via ArgSet so method is validated and "args" collected
		bound, err := as.BindArgs(interp, argv)
		if err != nil {
			as.ShowUsage(interp.Stderr)
			return EmptyToken, err
		}

		methodName := bound["method"].String
		m := methods[methodName]
		mt := m.Type()

		// Unpack positional tail as the method's arguments
		var userArgs []*Token
		if atoks, ok := bound["args"]; ok && atoks != nil {
			if lst, err := atoks.AsList(); err == nil {
				userArgs = lst
			} else {
				// if not a list, treat as single
				userArgs = []*Token{atoks}
			}
		}

		return invokeBoundMethod(m, mt, userArgs)
	})

	return &Token{
		String: fmt.Sprintf("%T", v),
		Data:   proc,
	}
}

func collectBoundMethods(recv reflect.Value) map[string]reflect.Value {
	methods := make(map[string]reflect.Value)
	addMethods := func(rv reflect.Value) {
		if !rv.IsValid() {
			return
		}
		rt := rv.Type()
		for i := 0; i < rt.NumMethod(); i++ {
			m := rt.Method(i)
			methods[m.Name] = rv.Method(i)
		}
	}

	addMethods(recv)
	if recv.CanAddr() {
		addMethods(recv.Addr())
	} else if recv.Kind() != reflect.Pointer {
		ptr := reflect.New(recv.Type())
		ptr.Elem().Set(recv)
		addMethods(ptr)
	}

	return methods
}

func convertTokenTo(tok *Token, dst reflect.Type) (reflect.Value, error) {
	// any / interface{}
	var srcIface any
	if tok.Data != nil {
		// Check if Data implements Interfacer() and use the result of that.
		if ier, ok := tok.Data.(Interfacer); ok {
			srcIface = ier.Interface()
		} else {
			srcIface = tok.Data
		}
	}

	if dst.Kind() == reflect.Interface && dst.NumMethod() == 0 {
		if tok.Data != nil {
			return reflect.ValueOf(srcIface), nil
		}
		return reflect.ValueOf(tok.String), nil
	}
	// prefer Data if assignable/convertible
	if tok.Data != nil {
		val := reflect.ValueOf(srcIface)

		if val.IsValid() && val.Type().AssignableTo(dst) {
			return val, nil
		}
		if val.IsValid() && val.Type().ConvertibleTo(dst) {
			return val.Convert(dst), nil
		}
	}
	switch dst.Kind() {
	case reflect.String:
		return reflect.ValueOf(tok.String).Convert(dst), nil
	case reflect.Bool:
		b, err := tok.AsBool()
		if err != nil {
			return reflect.Value{}, fmt.Errorf("expected bool, got %q", tok.String)
		}
		return reflect.ValueOf(b).Convert(dst), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(tok.String, 10, dst.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("expected integer, got %q", tok.String)
		}
		v := reflect.New(dst).Elem()
		v.SetInt(n)
		return v, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(tok.String, 10, dst.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("expected unsigned integer, got %q", tok.String)
		}
		v := reflect.New(dst).Elem()
		v.SetUint(n)
		return v, nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(tok.String, dst.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("expected float, got %q", tok.String)
		}
		v := reflect.New(dst).Elem()
		v.SetFloat(f)
		return v, nil
	case reflect.Interface:
		if tok.Data != nil {
			val := reflect.ValueOf(srcIface)
			if val.Type().AssignableTo(dst) {
				return val, nil
			}
			if val.Type().Implements(dst) {
				return val.Convert(dst), nil
			}
		}
	}
	if dst.Kind() == reflect.Struct ||
		(dst.Kind() == reflect.Pointer && dst.Elem().Kind() == reflect.Struct) ||
		dst.Kind() == reflect.Slice ||
		dst.Kind() == reflect.Map {
		if s := strings.TrimSpace(tok.String); s != "" {
			if norm, kind := InferJSON([]byte(s)); kind != jsonInvalid {
				target := reflect.New(dst)
				decodeInto := target.Interface()
				if dst.Kind() == reflect.Pointer {
					decodeInto = target.Elem().Interface()
				}
				if err := json.Unmarshal(norm, decodeInto); err == nil {
					if dst.Kind() == reflect.Pointer {
						return target.Elem(), nil
					}
					return target.Elem(), nil
				}
			}
		}
	}
	if dst.Kind() == reflect.Pointer {
		elem := dst.Elem()
		val, err := convertTokenTo(tok, elem)
		if err == nil {
			ptr := reflect.New(elem)
			ptr.Elem().Set(val)
			return ptr, nil
		}
	}
	return reflect.Value{}, fmt.Errorf("cannot convert %q to %s", tok.String, dst.String())
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "yes", "on":
		return true, nil
	case "0", "f", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("not a bool")
	}
}

// WrapDecider lets callers override auto-wrap behavior.
// Return true to wrap, false to leave as a plain NewToken.
var WrapDecider = func(t reflect.Type, v reflect.Value) bool {
	// default heuristic
	if !v.IsValid() {
		return false
	}

	// never wrap Token / *Token
	tokenT := reflect.TypeOf((*Token)(nil)).Elem()
	if t == tokenT || t == reflect.PointerTo(tokenT) {
		return false
	}

	// unwrap pointers for kind checks
	base := t
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}

	// basic scalars: don't wrap
	switch base.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return false
	}

	// wrap if it exposes ANY exported methods (value or pointer)
	if hasExportedMethod(v) {
		return true
	}
	if v.CanAddr() && hasExportedMethod(v.Addr()) {
		return true
	}

	return false
}

func hasExportedMethod(v reflect.Value) bool {
	t := v.Type()
	for i := 0; i < t.NumMethod(); i++ {
		// NumMethod only lists exported for non-package code, so >0 is enough.
		return true
	}
	return false
}

func wrapReturn(v any) *Token {
	if v == nil {
		return EmptyToken
	}
	switch x := v.(type) {
	case *Token:
		return x
	case Token:
		return &x
	}

	rv := reflect.ValueOf(v)
	rt := rv.Type()
	if WrapDecider(rt, rv) {
		return Wrap(v)
	}
	return NewToken(v)
}

func invokeBoundMethod(m reflect.Value, mt reflect.Type, userArgs []*Token) (*Token, error) {
	argc := mt.NumIn()
	if !mt.IsVariadic() {
		if len(userArgs) != argc {
			return EmptyToken, ErrArgCount(argc, len(userArgs))
		}
	} else {
		min := argc - 1
		if len(userArgs) < min {
			return EmptyToken, ErrArgMinimum(min, len(userArgs))
		}
	}

	capHint := argc
	if len(userArgs) > capHint {
		capHint = len(userArgs)
	}
	in := make([]reflect.Value, 0, capHint)
	if mt.IsVariadic() {
		for i := 0; i < argc-1; i++ {
			val, err := convertTokenTo(userArgs[i], mt.In(i))
			if err != nil {
				return EmptyToken, fmt.Errorf("arg %d: %w", i+1, err)
			}
			in = append(in, val)
		}
		for i := argc - 1; i < len(userArgs); i++ {
			val, err := convertTokenTo(userArgs[i], mt.In(argc-1).Elem())
			if err != nil {
				return EmptyToken, fmt.Errorf("arg %d: %w", i+1, err)
			}
			in = append(in, val)
		}
	} else {
		for i := 0; i < argc; i++ {
			val, err := convertTokenTo(userArgs[i], mt.In(i))
			if err != nil {
				return EmptyToken, fmt.Errorf("arg %d: %w", i+1, err)
			}
			in = append(in, val)
		}
	}

	out := m.Call(in)
	return wrapCallResults(out)
}

func wrapCallResults(out []reflect.Value) (*Token, error) {
	if n := len(out); n > 0 {
		if out[n-1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !out[n-1].IsNil() {
				return EmptyToken, out[n-1].Interface().(error)
			}
			out = out[:n-1]
		}
	}

	switch len(out) {
	case 0:
		return EmptyToken, nil
	case 1:
		return wrapReturn(out[0].Interface()), nil
	default:
		toks := make([]*Token, len(out))
		for i := range out {
			toks[i] = wrapReturn(out[i].Interface())
		}
		return NewList(toks), nil
	}
}
