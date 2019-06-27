// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles}
	pkgs, err := packages.Load(cfg,
		"cmd/gofmt",
		"golang.org/x/tools/cmd/goimports",

		// These are internal goimports dependencies. Copy them.
		"golang.org/x/tools/internal/imports",
		"golang.org/x/tools/internal/gopathwalk",
		"golang.org/x/tools/internal/module",
		"golang.org/x/tools/internal/fastwalk",
		"golang.org/x/tools/internal/semver",
	)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		var err error
		switch pkg.PkgPath {
		case "cmd/gofmt":
			err = copyGofmt(pkg.GoFiles)
		case "golang.org/x/tools/cmd/goimports":
			err = copyGoimports(pkg.GoFiles)
		case "golang.org/x/tools/internal/imports",
			"golang.org/x/tools/internal/gopathwalk",
			"golang.org/x/tools/internal/module",
			"golang.org/x/tools/internal/fastwalk",
			"golang.org/x/tools/internal/semver":
			parts := strings.Split(pkg.PkgPath, "/")
			dir := filepath.Join(append([]string{"gofumports"}, parts[3:]...)...)
			err = copyInternal(pkg.GoFiles, dir)
		default:
			return fmt.Errorf("unexpected package path %s", pkg.PkgPath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func copyGofmt(files []string) error {
	const extraSrc = `
		// This is the only gofumpt change on gofmt's codebase, besides changing
		// the name in the usage text.
		internal.Gofumpt(fileSet, file)
		`

	for _, fpath := range files {
		bodyBytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			return err
		}
		body := string(bodyBytes) // to simplify operations later
		name := filepath.Base(fpath)
		switch name {
		case "doc.go":
			continue // we have our own
		case "gofmt.go":
			i := strings.Index(body, "res, err := format(")
			if i < 0 {
				return fmt.Errorf("could not insert the gofumpt source code")
			}
			body = body[:i] + "\n" + extraSrc + "\n" + body[i:]
		}
		body = strings.Replace(body, "gofmt", "gofumpt", -1)
		if err := ioutil.WriteFile(name, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}

func copyGoimports(files []string) error {
	const extraSrc = `
		// This is the only gofumpt change on goimports's codebase, besides
		// changing the name in the usage text.
		res, err = internal.GofumptBytes(res)
		if err != nil {
			return err
		}
		`

	for _, fpath := range files {
		bodyBytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			return err
		}
		bodyBytes = fixImports(bodyBytes)
		body := string(bodyBytes) // to simplify operations later
		name := filepath.Base(fpath)
		switch name {
		case "doc.go":
			continue // we have our own
		case "goimports.go":
			i := strings.Index(body, "if !bytes.Equal")
			if i < 0 {
				return fmt.Errorf("could not insert the gofumports source code")
			}
			body = body[:i] + "\n" + extraSrc + "\n" + body[i:]
		}
		body = strings.Replace(body, "goimports", "gofumports", -1)

		dst := filepath.Join("gofumports", name)
		if err := ioutil.WriteFile(dst, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}

func copyInternal(files []string, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, fpath := range files {
		body, err := ioutil.ReadFile(fpath)
		if err != nil {
			return err
		}
		body = fixImports(body)

		name := filepath.Base(fpath)
		dst := filepath.Join(dir, name)
		if err := ioutil.WriteFile(dst, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}

func fixImports(body []byte) []byte {
	return bytes.Replace(body,
		[]byte("golang.org/x/tools/internal/"),
		[]byte("mvdan.cc/gofumpt/gofumports/internal/"),
		-1)
}
