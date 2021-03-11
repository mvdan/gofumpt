// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"gofumpt": func() int {
			gofumptMain()
			return exitCode
		},
	}))
}

var update = flag.Bool("u", false, "update testscript output files")

func TestScripts(t *testing.T) {
	t.Parallel()
	p := testscript.Params{
		Dir:           filepath.Join("testdata", "scripts"),
		UpdateScripts: *update,
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}
