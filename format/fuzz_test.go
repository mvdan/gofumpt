// Copyright (c) 2021, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package format

import (
	"errors"
	"fmt"
	"go/scanner"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
	"golang.org/x/tools/txtar"
)

func FuzzFormat(f *testing.F) {
	// Initialize the corpus with the Go files from our test scripts.
	paths, err := filepath.Glob(filepath.Join("..", "testdata", "script", "*.txtar"))
	qt.Assert(f, qt.IsNil(err))
	qt.Assert(f, qt.Not(qt.HasLen(paths, 0)))
	for _, path := range paths {
		archive, err := txtar.ParseFile(path)
		qt.Assert(f, qt.IsNil(err))
		for _, file := range archive.Files {
			f.Logf("adding %s from %s", file.Name, path)
			if strings.HasSuffix(file.Name, ".go") || strings.Contains(file.Name, ".go.") {
				f.Add(string(file.Data), int8(18), false) // -lang=go1.18
				f.Add(string(file.Data), int8(1), false)  // -lang=go1.1
				f.Add(string(file.Data), int8(18), true)  // -lang=go1.18 -extra
			}
		}
	}

	f.Fuzz(func(t *testing.T, src string,
		majorVersion int8, // Empty version if negative, 1.N otherwise.
		extraRules bool,
	) {
		// TODO: also fuzz Options.ModulePath
		opts := Options{ExtraRules: extraRules}
		if majorVersion >= 0 {
			opts.LangVersion = fmt.Sprintf("go1.%d", majorVersion)
		}

		orig := []byte(src)
		formatted, err := Source(orig, opts)
		if errors.As(err, &scanner.ErrorList{}) {
			return // invalid syntax from parsing
		}
		qt.Assert(t, qt.IsNil(err))
		_ = formatted

		// TODO: verify that the result is idempotent

		// TODO: verify that, if the input was valid Go 1.N syntax,
		// so is the output (how? go/parser lacks an option)

		// TODO: check calling format.Node directly as well

		qt.Assert(t, qt.Equals(string(orig), src),
			qt.Commentf("input source bytes were modified"))
	})
}
