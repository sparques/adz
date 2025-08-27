package adz

import (
	"encoding"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Token struct {
	String string
	Data   any
}

var (
	EmptyToken = &Token{}
	EmptyList  = []*Token{}
)

type TokenMarshaler interface {
	MarshalToken() (*Token, error)
}

type TokenUnmarshaler interface {
	UnmarshalToken(*Token) error
}

func NewToken(v any) *Token {
	if tm, ok := v.(TokenMarshaler); ok {
		tok, err := tm.MarshalToken()
		if err == nil {
			tok.Data = v
			return tok
		}
	}
	if tm, ok := v.(encoding.TextMarshaler); ok {
		buf, err := tm.MarshalText()
		if err == nil {
			return &Token{String: string(buf), Data: v}
		}
	}
	switch v := v.(type) {
	case *Token:
		return v
	case Token:
		return &v
	default:
		return &Token{
			String: fmt.Sprintf("%v", v),
			Data:   v,
		}
	}
}

func NewTokenString(str string) *Token {
	return &Token{
		String: str,
	}
}

func NewTokenInt(i int) *Token {
	return &Token{
		String: strconv.Itoa(i),
		Data:   i,
	}
}

func NewTokenListString(strs []string) (list []*Token) {
	for i := range strs {
		list = append(list, NewTokenString(strs[i]))
	}
	return
}

func NewTokenBytes(str []byte) *Token {
	return &Token{
		String: string(str),
	}
}

// NewTokenCat makes a new token by concatenating the supplied tokens together
func NewTokenCat(toks ...*Token) *Token {
	var catstr string
	for i := range toks {
		catstr += toks[i].String
	}
	return NewTokenString(catstr)
}

// TokenJoin joins together tokens Return string or return token??
func TokenJoin(toks []*Token, joinStr string) string {
	if len(toks) == 0 {
		return ""
	}
	builder := &strings.Builder{}
	builder.WriteString(toks[0].String)
	for _, str := range toks[1:] {
		builder.WriteString(joinStr)
		builder.WriteString(str.String)
	}
	return builder.String()
}

// Make Token implement error... Hmmm
func (tok *Token) Error() string {
	return tok.String
}

// Summary returns a string, sumarized with the middle bit elided
func (tok *Token) Summary() string {
	if len(tok.String) < 20 {
		return tok.String
	}

	return tok.String[:10] + "…" + tok.String[len(tok.String)-9:len(tok.String)]
}

// Quoted returns the string form of the token quoted (with {}) if needed.
// This only applies to backslashes and spaces
// TODO: add in data.(type) checks and rigorously quote?
func (tok *Token) Quoted() string {
	if strings.IndexAny(tok.String, "\\ \t\n") != -1 || len(tok.String) == 0 {
		return "{" + tok.String + "}"
	}
	return tok.String
}

// Literal is the converse of Quoted. It returns the token string
// stripped of any quoting brackets.
func (tok *Token) Literal() string {
	return stripLiteralBrackets(tok.String)
}

func (tok *Token) AsBool() (bool, error) {
	if val, ok := tok.Data.(bool); ok {
		return val, nil
	}
	switch strings.ToLower(tok.String) {
	case "true", "1", "on", "yes":
		tok.Data = true
	case "false", "0", "off", "no":
		tok.Data = false
	default:
		return false, fmt.Errorf("could not parse as bool value: %s", tok.String)
	}
	return tok.Data.(bool), nil
}

func (tok *Token) IsTrue() bool {
	b, _ := tok.AsBool()
	return b
}

func (tok *Token) AsInt() (int, error) {
	if val, ok := tok.Data.(int); ok {
		return val, nil
	}
	val, err := strconv.Atoi(tok.String)
	if err != nil {
		return 0, err
	}
	tok.Data = val
	return val, err
}

func (tok *Token) AsFloat() (float64, error) {
	if val, ok := tok.Data.(float64); ok {
		return val, nil
	}
	val, err := strconv.ParseFloat(tok.String, 64)
	if err != nil {
		return 0, err
	}
	tok.Data = val
	return val, err
}

func (tok *Token) AsScript() (Script, error) {
	// if already cached as script, just return it
	if script, ok := tok.Data.(Script); ok {
		return script, nil
	}
	// otherwise try to parse
	var err error
	tok.Data, err = LexString(tok.String)
	return tok.Data.(Script), err
}

func NewList(s []*Token) *Token {
	if len(s) == 0 {
		return EmptyToken
	}
	list := &Token{
		Data: s,
	}

	list.String = s[0].Quoted()

	for i := 1; i < len(s); i++ {
		list.String += " " + s[i].Quoted()
	}

	return list
}

type List []*Token

func (l List) Len() int {
	return len(l)
}

func (l List) Less(i, j int) bool {
	return l[i].String < l[j].String
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (tok *Token) AsList() (list []*Token, err error) {
	if list, ok := tok.Data.([]*Token); ok {
		return list, nil
	}
	if len(tok.String) == 0 {
		return EmptyList, nil
	}
	list, err = LexStringToList(tok.String)
	// don't overwrite tok.Data if there were an error
	if err == nil {
		tok.Data = list
	}
	return
}

// this might break my "variables are immutable" goal
func (tok *Token) Append(elements ...*Token) []*Token {
	list, err := tok.AsList()
	if err != nil {
		// if there were an error parsing this as a list,
		// treat it as a single element list
		list = append(list, tok)
	}
	list = append(list, elements...)
	return list
}

// ListOfOne returns true when interpreting token as
// a List, it has a Length of 1 or zero.
func (tok *Token) ListOfOne() bool {
	list, err := tok.AsList()

	if err != nil {
		return true
	}

	return len(list) < 2
}

// Index treats tok as a list and  returns the idx'th element of tok.
// A negative index is treated as backwards (so -1 is the last element).
// Non existent elements return an EmptyToken.
func (tok *Token) Index(idx int) *Token {
	list, err := tok.AsList()
	if err != nil {
		return EmptyToken
	}
	if idx < 0 {
		idx = len(list) + idx
	}

	if idx < 0 || idx >= len(list) {
		return EmptyToken
	}

	return list[idx]
}

// Len treats tok as a list and returns the number of elements in the list.
func (tok *Token) Len() int {
	list, err := tok.AsList()
	if err != nil {
		return 1
	}
	return len(list)
}

func (tok *Token) Slice(start, end int) *Token {
	list, err := tok.AsList()
	if err != nil {
		return EmptyToken
	}

	if start < 0 {
		start = len(list) + start
	}

	if start < 0 {
		start = 0
	}

	if end < 0 {
		end = len(list) + end
	}

	if end >= len(list) {
		end = len(list) - 1
	}

	if start > end {
		slice := list[end : start+1]
		slices.Reverse(slice)
		return NewList(slice)
	}

	return NewList(list[start : end+1])
}

func contains(s []*Token, e *Token) bool {
	for i := range s {
		if s[i].String == e.String {
			return true
		}
	}
	return false
}

func (tok *Token) Equal(c *Token) bool {
	// TODO, more thorough means of comparison.
	// Maybe check if .Data implements an Equal method?
	return tok.String == c.String
}
