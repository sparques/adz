package adz

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"reflect"
	"strings"
)

type FormatKind int

const (
	FormatAuto FormatKind = iota // default: fmt-style string or object’s own marshaler
	FormatJSON
	FormatJSONPretty
)

// per-package defaults; can be toggled globally if you want
var (
	DefaultFormat       = FormatAuto
	DefaultStrictJSON   = true // DisallowUnknownFields on overlay
	DefaultPrettyIndent = "  "
)

// GoObject wraps a *T and exposes methods/fields as a proc.
type GoObject struct {
	ptr reflect.Value // always *T
	typ reflect.Type  // T

	methods    map[string]reflect.Value
	fields     map[string]reflect.StructField
	methodSigs map[string]*ArgSet

	// formatting strategy
	Format            FormatKind
	StrictJSON        bool
	MarshalTextFunc   func() ([]byte, error)
	UnmarshalTextFunc func([]byte) error
}

// handy: expose underlying (pointer) for internal use
func (g *GoObject) Interface() any { return g.ptr.Interface() }

// MarshalText implements encoding.TextMarshaler.
func (g *GoObject) MarshalText() ([]byte, error) {
	// explicit override wins
	if g.MarshalTextFunc != nil {
		return g.MarshalTextFunc()
	}

	switch g.Format {
	case FormatJSON:
		return json.Marshal(g.Interface())
	case FormatJSONPretty:
		return json.MarshalIndent(g.Interface(), "", DefaultPrettyIndent)
	case FormatAuto:
		// If the underlying object implements its own TextMarshaler,
		// honor it unless you really want to force your own format.
		if tm, ok := g.ptr.Interface().(encoding.TextMarshaler); ok {
			return tm.MarshalText()
		}
		// Fallback to fmt
		return []byte(fmt.Sprintf("%v", g.Interface())), nil
	default:
		return []byte(fmt.Sprintf("%v", g.Interface())), nil
	}
}

// UnmarshalText implements encoding.TextUnmarshaler (used for overlays or explicit set).
func (g *GoObject) UnmarshalText(p []byte) error {
	// explicit override wins
	if g.UnmarshalTextFunc != nil {
		return g.UnmarshalTextFunc(p)
	}

	// If underlying has its own TextUnmarshaler, let it handle it.
	if tu, ok := g.ptr.Interface().(encoding.TextUnmarshaler); ok {
		return tu.UnmarshalText(p)
	}

	// Default is JSON (strict vs non-strict)
	dec := json.NewDecoder(bytes.NewReader(p))
	if g.StrictJSON {
		dec.DisallowUnknownFields()
	}
	return dec.Decode(g.ptr.Interface())
}

type WrapOption func(*GoObject)

func WithFormat(kind FormatKind) WrapOption {
	return func(g *GoObject) { g.Format = kind }
}
func WithStrictJSON(strict bool) WrapOption {
	return func(g *GoObject) { g.StrictJSON = strict }
}
func WithTextFuncs(m func() ([]byte, error), u func([]byte) error) WrapOption {
	return func(g *GoObject) {
		g.MarshalTextFunc = m
		g.UnmarshalTextFunc = u
	}
}

func WrapObject(v any, sigs map[string]*ArgSet, opts ...WrapOption) *Token {
	goObj := newGoObject(v, sigs)
	// defaults
	goObj.Format = DefaultFormat
	goObj.StrictJSON = DefaultStrictJSON
	// apply overrides
	for _, opt := range opts {
		opt(goObj)
	}
	t := NewTokenString(goObj.typ.PkgPath() + "." + goObj.typ.Name())
	t.Data = Procer(goObj) // important: invocable
	return t
}

func ProcObject[OBJ any](interp *Interp, args []*Token) (*Token, error) {
	as := NewArgSet(args[0].String,
		ArgFull("-format", NewToken("auto"), NewToken(`tuple {auto json json-pretty}`),
			"How to render the token's string."),
		ArgFull("-strict", NewToken("true"), NewToken(`bool`),
			"Disallow unknown fields in JSON."),
		ArgHelp("var", "variable to create/update"),
		ArgDefaultHelp("json", NewToken(""), "optional JSON body or array"),
	)
	bound, err := as.BindArgs(interp, args)
	if err != nil {
		as.ShowUsage(interp.Stderr)
		return EmptyToken, err
	}

	tOBJ := typeOfT[OBJ]() // the compile-time OBJ type (could be T or *T)
	var tok *Token         // existing var, if any
	varName := bound["var"].String

	// Try to load existing var (ok if missing)
	if v, err := interp.GetVar(varName); err == nil {
		tok = v
	} else if !errors.Is(err, ErrNoVar) {
		return EmptyToken, err
	}

	// Build exactly *T for wrapping, regardless of OBJ being T or *T
	var ptrToStruct reflect.Value
	switch {
	case tOBJ.Kind() == reflect.Ptr && tOBJ.Elem().Kind() == reflect.Struct:
		// OBJ is *T → allocate *T
		ptrToStruct = reflect.New(tOBJ.Elem())
	case tOBJ.Kind() == reflect.Struct:
		// OBJ is T → allocate *T
		ptrToStruct = reflect.New(tOBJ)
	default:
		return EmptyToken, fmt.Errorf("ProcObject expects OBJ to be T or *T where T is a struct; got %v", tOBJ)
	}

	// Seed from existing token if it already holds OBJ or compatible JSON
	if tok != nil {
		if tok.Data != nil {
			rv := reflect.ValueOf(tok.Data)
			// If it’s T, copy into *T
			if rv.Type() == ptrToStruct.Elem().Type() {
				ptrToStruct.Elem().Set(rv)
			} else if rv.Type() == ptrToStruct.Type() {
				// already *T
				ptrToStruct.Elem().Set(rv.Elem())
			}
		} else if s := strings.TrimSpace(tok.String); s != "" {
			if norm, k := InferJSON([]byte(s)); k != jsonInvalid {
				if err := json.Unmarshal(norm, ptrToStruct.Interface()); err != nil {
					return EmptyToken, err
				}
			}
		}
	}

	// Optional overlay from -json (strict by default)
	if js := strings.TrimSpace(bound["json"].String); js != "" {
		norm, k := InferJSON([]byte(js))
		if k == jsonInvalid {
			return EmptyToken, fmt.Errorf("invalid JSON for -json")
		}
		dec := json.NewDecoder(bytes.NewReader(norm))
		if strings.EqualFold(bound["-strict"].String, "true") {
			dec.DisallowUnknownFields()
		}
		if err := dec.Decode(ptrToStruct.Interface()); err != nil {
			return EmptyToken, err
		}
	}

	// Choose default string format for *this instance*
	var fmtKind FormatKind
	switch strings.ToLower(bound["-format"].String) {
	case "json":
		fmtKind = FormatJSON
	case "json-pretty":
		fmtKind = FormatJSONPretty
	default:
		fmtKind = FormatAuto
	}

	// Wrap EXACTLY *T (not **T)
	wrapped := WrapObject(ptrToStruct.Interface(), nil,
		WithFormat(fmtKind),
		WithStrictJSON(strings.EqualFold(bound["-strict"].String, "true")),
	)
	return interp.SetVar(varName, wrapped)
}

func typeOfT[OBJ any]() reflect.Type {
	var z *OBJ
	return reflect.TypeOf(z).Elem() // the "OBJ" type itself
}

func newGoObject(v any, methodSigs map[string]*ArgSet) *GoObject {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		panic("nil object")
	}
	// ensure pointer
	if rv.Kind() != reflect.Ptr {
		// if addressable, take address; else copy into new *T
		if rv.CanAddr() {
			rv = rv.Addr()
		} else {
			dst := reflect.New(rv.Type())
			dst.Elem().Set(rv)
			rv = dst
		}
	}
	rt := rv.Elem().Type() // T

	// collect methods on pointer (full set)
	m := map[string]reflect.Value{}
	ptrType := rv.Type()
	for i := 0; i < ptrType.NumMethod(); i++ {
		mt := ptrType.Method(i)
		if token.IsExported(mt.Name) {
			m[mt.Name] = rv.Method(i) // bound method
		}
	}

	// collect struct fields (exported only)
	f := map[string]reflect.StructField{}
	if rt.Kind() == reflect.Struct {
		for i := 0; i < rt.NumField(); i++ {
			sf := rt.Field(i)
			if sf.PkgPath == "" { // exported
				f[sf.Name] = sf
			}
		}
	}

	return &GoObject{
		ptr:        rv,
		typ:        rt,
		methods:    m,
		fields:     f,
		methodSigs: methodSigs,
	}
}

// Procer: `$obj <thing> [args...]`
func (g *GoObject) Proc(interp *Interp, args []*Token) (*Token, error) {
	if len(args) < 2 {
		return EmptyToken, ErrArgMinimum(1, 0)
	}
	name := args[1].String

	// 1) conventional help: "$obj help" or "$obj help Method"
	if name == "help" {
		if len(args) == 2 {
			return EmptyToken, nil
			// return NewTokenString(g.helpOverview()), nil
		}
		target := args[2].String
		if as, ok := g.methodSigs[target]; ok {
			return NewTokenString(as.HelpText()), nil
		}
		if _, ok := g.fields[target]; ok {
			return EmptyToken, nil
			// return NewTokenString(g.helpField(target)), nil
		}
		return EmptyToken, ErrCommand(fmt.Sprintf("no such method/field %q", target))
	}

	// 2) dot-prefix means field access: "$obj .Field [newValue?]"
	if strings.HasPrefix(name, ".") {
		field := strings.TrimPrefix(name, ".")
		sf, ok := g.fields[field]
		if !ok {
			return EmptyToken, ErrCommand(fmt.Sprintf("no such field %q", field))
		}
		v := g.ptr.Elem().FieldByIndex(sf.Index)
		switch len(args) {
		case 2: // get
			return wrapReturn(v.Interface()), nil
		case 3: // set
			newVal, err := coerceTokenTo(interp, args[2], v.Type())
			if err != nil {
				return EmptyToken, fmt.Errorf("field %s: %w", field, err)
			}
			if !v.CanSet() {
				return EmptyToken, fmt.Errorf("field %s is not settable", field)
			}
			v.Set(reflect.ValueOf(newVal))
			return wrapReturn(v.Interface()), nil
		default:
			return EmptyToken, ErrArgCount(1, len(args)-2)
		}
	}

	// 3) otherwise: method call
	if meth, ok := g.methods[name]; ok {
		// optional: ArgSet validation if present
		if as, ok := g.methodSigs[name]; ok {
			if _, err := as.BindArgs(interp, append([]*Token{args[1]}, args[2:]...)); err != nil {
				as.ShowUsage(interp.Stderr)
				return EmptyToken, err
			}
		}

		callArgs := make([]reflect.Value, meth.Type().NumIn())
		// Start consuming from args[2:]
		user := args[2:]
		if len(user) != len(callArgs) {
			return EmptyToken, ErrArgCount(len(callArgs), len(user))
		}
		for i := range callArgs {
			goVal, err := coerceTokenTo(interp, user[i], meth.Type().In(i))
			if err != nil {
				return EmptyToken, fmt.Errorf("%s arg %d: %w", name, i+1, err)
			}
			callArgs[i] = reflect.ValueOf(goVal)
		}
		out := meth.Call(callArgs)
		// error as last result?
		if n := len(out); n > 0 {
			if errT := out[n-1].Type(); errT.Name() == "error" && errT.PkgPath() == "" {
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

	return EmptyToken, ErrCommand(fmt.Sprintf("no such method/field %q", name))
}

func coerceTokenTo(interp *Interp, tok *Token, want reflect.Type) (any, error) {
	// fast-path: exact dynamic type already matches
	if tok.Data != nil && reflect.TypeOf(tok.Data).AssignableTo(want) {
		return tok.Data, nil
	}
	// string → try JSON into composite
	if want.Kind() == reflect.Struct || (want.Kind() == reflect.Ptr && want.Elem().Kind() == reflect.Struct) ||
		want.Kind() == reflect.Slice || want.Kind() == reflect.Map {
		if s := strings.TrimSpace(tok.String); s != "" {
			if norm, kind := InferJSON([]byte(s)); kind != jsonInvalid {
				dst := reflect.New(want).Interface()
				if want.Kind() != reflect.Ptr {
					dst = reflect.New(want).Interface()
				}
				if err := json.Unmarshal(norm, dst); err == nil {
					if want.Kind() == reflect.Ptr {
						return reflect.ValueOf(dst).Elem().Interface(), nil // want *T, dst is *T
					}
					return reflect.ValueOf(dst).Elem().Elem().Interface(), nil // want T
				}
			}
		}
	}
	// primitives: int/bool/float/string
	switch want.Kind() {
	case reflect.String:
		return tok.String, nil
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		i, err := tok.AsInt()
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(i).Convert(want).Interface(), nil
	case reflect.Bool:
		b, err := tok.AsBool()
		if err != nil {
			return nil, err
		}
		return b, nil
	case reflect.Float32, reflect.Float64:
		f, err := tok.AsFloat()
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(f).Convert(want).Interface(), nil
	}
	// fallback: pass Data if any
	if tok.Data != nil && reflect.TypeOf(tok.Data).ConvertibleTo(want) {
		return reflect.ValueOf(tok.Data).Convert(want).Interface(), nil
	}
	return nil, fmt.Errorf("cannot coerce %q (%T) to %v", tok.String, tok.Data, want)
}

const (
	jsonObj = iota
	jsonArray
	jsonInvalid
)

// InferJSON normalizes data into a valid JSON object or array.
// It returns (normalizedJSON, kind). If normalization is impossible,
// it returns (nil, jsonInvalid).
//
// Accepted forms:
//   - Proper JSON object:        `{ ... }`
//   - Proper JSON array:         `[ ... ]`
//   - Object body (no braces):   `key: "v", "k2": 3`   → wrapped as `{ ... }` (strict JSON required, keys quoted)
//   - Array body (no brackets):  `"a", 2, true`        → wrapped as `[ ... ]`
func InferJSON(data []byte) ([]byte, int) {
	s := bytes.TrimSpace(data)
	if len(s) == 0 {
		return nil, jsonInvalid
	}

	switch s[0] {
	case '{':
		if lastNonSpace(s) == '}' && validJSONMap(s) {
			return s, jsonObj
		}
		return nil, jsonInvalid
	case '[':
		if lastNonSpace(s) == ']' && validJSONArray(s) {
			return s, jsonArray
		}
		return nil, jsonInvalid
	}

	// No outer delimiters; sniff to bias the wrap.
	switch sniffKindNoDelims(s) {
	case jsonObj:
		w := wrap(s, '{', '}')
		if validJSONMap(w) {
			return w, jsonObj
		}
	case jsonArray:
		w := wrap(s, '[', ']')
		if validJSONArray(w) {
			return w, jsonArray
		}
	default:
		// Ambiguous → try both.
		if w := wrap(s, '{', '}'); validJSONMap(w) {
			return w, jsonObj
		}
		if w := wrap(s, '[', ']'); validJSONArray(w) {
			return w, jsonArray
		}
	}

	return nil, jsonInvalid
}

func lastNonSpace(s []byte) byte {
	i := len(s) - 1
	for i >= 0 && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\f') {
		i--
	}
	if i < 0 {
		return 0
	}
	return s[i]
}

func validJSONMap(s []byte) bool {
	var m map[string]any
	return json.Unmarshal(s, &m) == nil
}
func validJSONArray(s []byte) bool {
	var a []any
	return json.Unmarshal(s, &a) == nil
}

func wrap(s []byte, open, close byte) []byte {
	out := make([]byte, 0, len(s)+2)
	out = append(out, open)
	out = append(out, s...)
	out = append(out, close)
	return out
}

// sniffKindNoDelims does a light structural pass:
// - tracks strings/escapes
// - counts nesting of {,[ … ],}
// - returns jsonObj if it sees ':' at depth 0
// - returns jsonArray if it sees ',' at depth 0
// - otherwise jsonInvalid (ambiguous → caller tries both wraps)
func sniffKindNoDelims(s []byte) int {
	inStr := false
	esc := false
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return jsonObj
			}
		case ',':
			if depth == 0 {
				return jsonArray
			}
		}
	}
	return jsonInvalid
}
