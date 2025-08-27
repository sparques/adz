package adz

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

func init() {
	StdLib["field"] = procField
}

// procField implements the field proc which extracts values out of an
// object by glob matching to a dot-delineated key.
// output is an List
func procField(interp *Interp, args []*Token) (*Token, error) {
	// TODO: check args :)

	namedProto, posProto, err := ParseProto(NewToken(`{-values true} {-keys false} {-separator .} {-matchcase false} obj args`))
	if err != nil {
		return EmptyToken, err
	}

	parsedArgs, err := ParseArgs(namedProto, posProto, args[1:])
	if err != nil {
		return EmptyToken, err
	}

	var obj any = parsedArgs["obj"].Data

	if obj == nil {
		// what is a sane default? Error? I think for now we'll split on new lines
		obj = strings.Split(parsedArgs["obj"].String, "\n")
	}

	objmap := flatten(obj, flattenOption{
		Sep: parsedArgs["separator"].String,
	})

	keep := map[string]any{}

	patterns, _ := parsedArgs["args"].AsList()
	for i, p := range patterns {
		if !parsedArgs["matchcase"].IsTrue() {
			p.String = strings.ToLower(p.String)
		}
		for k, v := range objmap {
			if !parsedArgs["matchcase"].IsTrue() {
				k = strings.ToLower(k)
			}
			match, err := filepath.Match(p.String, k)
			if err != nil {
				return EmptyToken, fmt.Errorf("arg %d: glob pattern: %w", i+1, err)
			}
			if match {
				keep[k] = v
			}
		}
	}

	out := []*Token{}
	for k, v := range keep {
		if parsedArgs["keys"].IsTrue() {
			out = append(out, NewToken(k))
		}
		if parsedArgs["values"].IsTrue() {
			out = append(out, NewToken(v))
		}
	}

	return NewList(out), nil
}

type flattenOption struct {
	Tag         string // e.g. "json" (default)
	Sep         string // path separator; default "."
	Root        string // optional root prefix
	AllowInline bool   // if true, treat tag option "inline" (yaml-style) as path-flattening
}

func flatten(v any, opt flattenOption) map[string]any {
	if opt.Tag == "" {
		opt.Tag = "json"
	}
	if opt.Sep == "" {
		opt.Sep = "."
	}

	out := make(map[string]any)
	seen := make(map[uintptr]struct{})

	var rec func(path string, rv reflect.Value)
	rec = func(path string, rv reflect.Value) {
		// unwrap pointers/interfaces
		for rv.IsValid() && (rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer) {
			if rv.IsNil() {
				if path != "" {
					out[path] = nil
				}
				return
			}
			rv = rv.Elem()
		}
		if !rv.IsValid() {
			if path != "" {
				out[path] = nil
			}
			return
		}

		// treat common special scalars as atomic
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			if path != "" {
				out[path] = rv.Interface()
			}
			return
		}

		// special case of an *Token
		if tok, ok := rv.Interface().(Token); ok {
			if tok.Data != nil {
				rv = reflect.ValueOf(tok.Data)
			} else {
				rv = reflect.ValueOf(tok.String)
			}
		}

		switch rv.Kind() {
		case reflect.Struct:

			// cycle guard (only if addressable)
			if rv.CanAddr() {
				ptr := rv.Addr().Pointer()
				if _, ok := seen[ptr]; ok {
					return
				}
				seen[ptr] = struct{}{}
			}
			t := rv.Type()
			for i := 0; i < rv.NumField(); i++ {
				sf := t.Field(i)
				if sf.PkgPath != "" { // unexported
					continue
				}
				name, inline, skip := parseTag(sf, opt.Tag, opt.AllowInline)
				if skip {
					continue
				}
				childPath := path
				if !inline {
					if name == "" {
						name = sf.Name
					}
					childPath = join(path, name, opt.Sep)
				}
				rec(childPath, rv.Field(i))
			}
		case reflect.Map:
			if rv.IsNil() {
				if path != "" {
					out[path] = nil
				}
				return
			}
			// maps have a stable pointer identity
			ptr := rv.Pointer()
			if ptr != 0 {
				if _, ok := seen[ptr]; ok {
					return
				}
				seen[ptr] = struct{}{}
			}
			for _, k := range rv.MapKeys() {
				key := fmt.Sprint(k.Interface())
				rec(join(path, key, opt.Sep), rv.MapIndex(k))
			}
		case reflect.Slice, reflect.Array:
			if rv.Kind() == reflect.Slice && rv.IsNil() {
				if path != "" {
					out[path] = nil
				}
				return
			}
			// cycle guard for slices
			if rv.Kind() == reflect.Slice {
				ptr := rv.Pointer()
				if ptr != 0 {
					if _, ok := seen[ptr]; ok {
						return
					}
					seen[ptr] = struct{}{}
				}
			}
			for i := 0; i < rv.Len(); i++ {
				rec(fmt.Sprintf("%s%s%d", path, opt.Sep, i), rv.Index(i))
			}
		default:
			if path != "" {
				out[path] = rv.Interface()
			}
		}
	}

	root := strings.TrimSpace(opt.Root)
	if root != "" {
		rec(root, reflect.ValueOf(v))
	} else {
		rec("", reflect.ValueOf(v))
		// normalize any accidental leading separators (defensive)
		if opt.Sep != "" {
			for k, val := range out {
				if strings.HasPrefix(k, opt.Sep) {
					out[strings.TrimPrefix(k, opt.Sep)] = val
					delete(out, k)
				}
			}
		}
	}
	return out
}

func parseTag(sf reflect.StructField, tag string, allowInline bool) (name string, inline, skip bool) {
	if tag == "" {
		return "", false, false
	}
	tagVal, ok := sf.Tag.Lookup(tag)
	if !ok {
		return "", false, false
	}
	if tagVal == "-" {
		return "", false, true
	}
	parts := strings.Split(tagVal, ",")
	if len(parts) == 0 {
		return "", false, false
	}
	name = parts[0] // may be ""
	for _, opt := range parts[1:] {
		if allowInline && opt == "inline" {
			inline = true
		}
	}
	return name, inline, false
}

func join(base, elem, sep string) string {
	switch {
	case base == "":
		return elem
	case elem == "":
		return base
	default:
		return base + sep + elem
	}
}
