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
