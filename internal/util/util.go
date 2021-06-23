// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package util implements a series of helpers to manipulate AST fields.

package util

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Values/types for special cases.
var (
	objectPtrNil = reflect.ValueOf((*ast.Object)(nil))
	scopePtrNil  = reflect.ValueOf((*ast.Scope)(nil))

	identType     = reflect.TypeOf((*ast.Ident)(nil))
	objectPtrType = reflect.TypeOf((*ast.Object)(nil))
	positionType  = reflect.TypeOf(token.NoPos)
	callExprType  = reflect.TypeOf((*ast.CallExpr)(nil))
	scopePtrType  = reflect.TypeOf((*ast.Scope)(nil))
)

// set is a wrapper for x.Set(y); it protects the caller from panics if x cannot be changed to y.
func set(x, y reflect.Value) {
	// don't bother if x cannot be set or y is invalid
	if !x.CanSet() || !y.IsValid() {
		return
	}
	defer func() {
		if x := recover(); x != nil {
			if s, ok := x.(string); ok &&
				(strings.Contains(s, "type mismatch") || strings.Contains(s, "not assignable")) {
				// x cannot be set to y - ignore this rewrite
				return
			}
			panic(x)
		}
	}()
	x.Set(y)
}

func isWildcard(s string) bool {
	rune, size := utf8.DecodeRuneInString(s)
	return size == len(s) && unicode.IsLower(rune)
}

// Apply replaces each AST field x in val with f(x), returning val.
// To avoid extra conversions, f operates on the reflect.Value form.
func Apply(f func(reflect.Value) reflect.Value, val reflect.Value) reflect.Value {
	if !val.IsValid() {
		return reflect.Value{}
	}

	// *ast.Objects introduce cycles and are likely incorrect after
	// rewrite; don't follow them but replace with nil instead
	if val.Type() == objectPtrType {
		return objectPtrNil
	}

	// similarly for scopes: they are likely incorrect after a rewrite;
	// replace them with nil
	if val.Type() == scopePtrType {
		return scopePtrNil
	}

	switch v := reflect.Indirect(val); v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			set(e, f(e))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			e := v.Field(i)
			set(e, f(e))
		}
	case reflect.Interface:
		e := v.Elem()
		set(v, f(e))
	}
	return val
}

// Match reports whether pattern matches val,
// recording wildcard submatches in m.
// If m == nil, match checks whether pattern == val.
func Match(m map[string]reflect.Value, pattern, val reflect.Value) bool {
	// Wildcard matches any expression. If it appears multiple
	// times in the pattern, it must match the same expression
	// each time.
	if m != nil && pattern.IsValid() && pattern.Type() == identType {
		name := pattern.Interface().(*ast.Ident).Name
		if isWildcard(name) && val.IsValid() {
			// wildcards only match valid (non-nil) expressions.
			if _, ok := val.Interface().(ast.Expr); ok && !val.IsNil() {
				if old, ok := m[name]; ok {
					return Match(nil, old, val)
				}
				m[name] = val
				return true
			}
		}
	}

	// Otherwise, pattern and val must match recursively.
	if !pattern.IsValid() || !val.IsValid() {
		return !pattern.IsValid() && !val.IsValid()
	}
	if pattern.Type() != val.Type() {
		return false
	}

	// Special cases.
	switch pattern.Type() {
	case identType:
		// For identifiers, only the names need to match
		// (and none of the other *ast.Object information).
		// This is a common case, handle it all here instead
		// of recursing down any further via reflection.
		p := pattern.Interface().(*ast.Ident)
		v := val.Interface().(*ast.Ident)
		return p == nil && v == nil || p != nil && v != nil && p.Name == v.Name
	case objectPtrType, positionType:
		// object pointers and token positions always match
		return true
	case callExprType:
		// For calls, the Ellipsis fields (token.Position) must
		// match since that is how f(x) and f(x...) are different.
		// Check them here but fall through for the remaining fields.
		p := pattern.Interface().(*ast.CallExpr)
		v := val.Interface().(*ast.CallExpr)
		if p.Ellipsis.IsValid() != v.Ellipsis.IsValid() {
			return false
		}
	}

	p := reflect.Indirect(pattern)
	v := reflect.Indirect(val)
	if !p.IsValid() || !v.IsValid() {
		return !p.IsValid() && !v.IsValid()
	}

	switch p.Kind() {
	case reflect.Slice:
		if p.Len() != v.Len() {
			return false
		}
		for i := 0; i < p.Len(); i++ {
			if !Match(m, p.Index(i), v.Index(i)) {
				return false
			}
		}
		return true

	case reflect.Struct:
		for i := 0; i < p.NumField(); i++ {
			if !Match(m, p.Field(i), v.Field(i)) {
				return false
			}
		}
		return true

	case reflect.Interface:
		return Match(m, p.Elem(), v.Elem())
	}

	// Handle token integers, etc.
	return p.Interface() == v.Interface()
}

// Subst returns a copy of pattern with values from m substituted in place
// of wildcards and pos used as the position of tokens from the pattern.
// if m == nil, subst returns a copy of pattern and doesn't change the line
// number information.
func Subst(m map[string]reflect.Value, pattern reflect.Value, pos reflect.Value) reflect.Value {
	if !pattern.IsValid() {
		return reflect.Value{}
	}

	// Wildcard gets replaced with map value.
	if m != nil && pattern.Type() == identType {
		name := pattern.Interface().(*ast.Ident).Name
		if isWildcard(name) {
			if old, ok := m[name]; ok {
				return Subst(nil, old, reflect.Value{})
			}
		}
	}

	if pos.IsValid() && pattern.Type() == positionType {
		// use new position only if old position was valid in the first place
		if old := pattern.Interface().(token.Pos); !old.IsValid() {
			return pattern
		}
		return pos
	}

	// Otherwise copy.
	switch p := pattern; p.Kind() {
	case reflect.Slice:
		if p.IsNil() {
			// Do not turn nil slices into empty slices. go/ast
			// guarantees that certain lists will be nil if not
			// populated.
			return reflect.Zero(p.Type())
		}
		v := reflect.MakeSlice(p.Type(), p.Len(), p.Len())
		for i := 0; i < p.Len(); i++ {
			v.Index(i).Set(Subst(m, p.Index(i), pos))
		}
		return v

	case reflect.Struct:
		v := reflect.New(p.Type()).Elem()
		for i := 0; i < p.NumField(); i++ {
			v.Field(i).Set(Subst(m, p.Field(i), pos))
		}
		return v

	case reflect.Ptr:
		v := reflect.New(p.Type()).Elem()
		if elem := p.Elem(); elem.IsValid() {
			v.Set(Subst(m, elem, pos).Addr())
		}
		return v

	case reflect.Interface:
		v := reflect.New(p.Type()).Elem()
		if elem := p.Elem(); elem.IsValid() {
			v.Set(Subst(m, elem, pos))
		}
		return v
	}

	return pattern
}
