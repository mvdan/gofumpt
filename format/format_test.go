// Copyright (c) 2021, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package format_test

import (
	"testing"

	"github.com/go-quicktest/qt"

	"mvdan.cc/gofumpt/format"
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
	got, err := format.Source(in, format.Options{})
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(got), string(want)))
}

func TestSourceSortsImportsFirst(t *testing.T) {
	t.Parallel()

	// The rules never move a commented import, so the std import is only at
	// the top if the imports are sorted before the rules, as the CLI does.
	in := []byte(`
package p

import (
	"zz.dev/q"
	"go/ast" // c
)
`[1:])
	want := []byte(`
package p

import (
	"go/ast" // c

	"zz.dev/q"
)
`[1:])
	got, err := format.Source(in, format.Options{})
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(got), string(want)))
}
