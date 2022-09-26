// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	qt "github.com/frankban/quicktest"
	exec "golang.org/x/sys/execabs"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"gofumpt": main1,
	}))
}

var update = flag.Bool("u", false, "update testscript output files")

func TestScript(t *testing.T) {
	t.Parallel()

	var goEnv struct {
		GOCACHE    string
		GOMODCACHE string
		GOMOD      string
	}
	out, err := exec.Command("go", "env", "-json").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &goEnv); err != nil {
		t.Fatal(err)
	}

	p := testscript.Params{
		Dir:                 filepath.Join("testdata", "script"),
		UpdateScripts:       *update,
		RequireExplicitExec: true,
		Setup: func(env *testscript.Env) error {
			env.Setenv("GOCACHE", goEnv.GOCACHE)
			env.Setenv("GOMODCACHE", goEnv.GOMODCACHE)
			env.Setenv("GOMOD_DIR", filepath.Dir(goEnv.GOMOD))
			return nil
		},
	}
	err = gotooltest.Setup(&p)
	qt.Assert(t, err, qt.IsNil)
	testscript.Run(t, p)
}
