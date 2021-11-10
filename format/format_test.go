// Copyright (c) 2021, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package format

import (
	"go/ast"
	"go/token"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestSourceIncludesSimplify(t *testing.T) {
	t.Parallel()

	in := []byte(`
package p

var ()

func f() {
	for _ = range v {
	}
}
`[1:])
	want := []byte(`
package p

func f() {
	for range v {
	}
}
`[1:])
	got, err := Source(in, Options{})
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, string(got), qt.Equals, string(want))
}

func TestIsCgoImport(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		desc  string
		value string
		want  bool
	}{
		{"backquoted", "`C`", true},
		{"double-quoted", "\"C\"", true},
		{"bad quote syntax", "\"C", false},
		{"not cgo import", "\"fmt\"", false},
	}

	for _, tt := range testcases {
		t.Run(tt.desc, func(t *testing.T) {
			decl := &ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{Value: tt.value},
					},
				},
			}
			qt.Assert(t, isCgoImport(decl), qt.Equals, tt.want)
		})
	}
}
