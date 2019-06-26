// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// +build ignore

package main

import (
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
	)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		switch pkg.PkgPath {
		case "cmd/gofmt":
			copyGofmt(pkg.GoFiles)
		case "golang.org/x/tools/cmd/goimports":
			copyGoimports(pkg.GoFiles)
		default:
			return fmt.Errorf("unexpected package path %s", pkg.PkgPath)
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

	for _, path := range files {
		bodyBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(bodyBytes) // to simplify operations later
		name := filepath.Base(path)
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

	for _, path := range files {
		bodyBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(bodyBytes) // to simplify operations later
		name := filepath.Base(path)
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
