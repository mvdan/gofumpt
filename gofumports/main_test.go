// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"gofumpt": func() int {
			// Don't change gofmtMain, to keep changes to the gofmt
			// codebase to a minimum.
			gofmtMain()
			return exitCode
		},
	}))
}

func TestScripts(t *testing.T) {
	t.Parallel()
	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("..", "testdata", "scripts"),
		Condition: func(cond string) (bool, error) {
			switch cond {
			case "gofumports":
				return true, nil
			}
			return false, nil
		},
	})
}
