// Copyright (c) 2021, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package format_test

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"mvdan.cc/gofumpt/format"
)

func TestSourceIncludesSimplify(t *testing.T) {
	t.Parallel()

	in := []byte(`
package p

var ()

func f() {
	make([]int, len, len)
	make([]int, 0)
	make(chan int, 0)
	make(map[int]int, 0)
	for _ = range v {
	}
}
`[1:])
	want := []byte(`
package p

func f() {
	make([]int, len)
	make([]int, 0)
	make(chan int)
	make(map[int]int)
	for range v {
	}
}
`[1:])
	got, err := format.Source(in, format.Options{})
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, string(got), qt.Equals, string(want))
}
