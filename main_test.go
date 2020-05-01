// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"mvdan.cc/gofumpt/internal"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"gofumpt": func() int {
			gofumptMain()
			return exitCode
		},
	}))
}

func TestScripts(t *testing.T) {
	t.Parallel()
	for _, dir := range internal.TestscriptDirs {
		testscript.Run(t, testscript.Params{
			Dir: dir,
			Condition: func(cond string) (bool, error) {
				switch cond {
				case "gofumpt":
					return true, nil
				}
				return false, nil
			},
		})
	}
}
