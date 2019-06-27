// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// +build ignore

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
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
		panic(err)
	}
	for _, pkg := range pkgs {
		switch pkg.PkgPath {
		case "cmd/gofmt":
			copyGofmt(pkg.GoFiles)
		case "golang.org/x/tools/cmd/goimports":
			copyGoimports(pkg.GoFiles)
		default:
			parts := strings.Split(pkg.PkgPath, "/")
			dir := filepath.Join(append([]string{"gofumports"}, parts[3:]...)...)
			copyInternal(pkg.GoFiles, dir)
		}
	}
}

func readFile(path string) string {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(body)
}

func writeFile(path, body string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(path, []byte(body), 0644); err != nil {
		panic(err)
	}
}

func copyGofmt(files []string) {
	const extraSrc = `
		// This is the only gofumpt change on gofmt's codebase, besides changing
		// the name in the usage text.
		internal.Gofumpt(fileSet, file)
		`
	for _, path := range files {
		body := readFile(path)
		name := filepath.Base(path)
		switch name {
		case "doc.go":
			continue // we have our own
		case "gofmt.go":
			i := strings.Index(body, "res, err := format(")
			if i < 0 {
				panic("could not insert the gofumpt source code")
			}
			body = body[:i] + "\n" + extraSrc + "\n" + body[i:]
		}
		body = strings.Replace(body, "gofmt", "gofumpt", -1)
		writeFile(name, body)
	}
}

func copyGoimports(files []string) {
	const extraSrc = `
		// This is the only gofumpt change on goimports's codebase, besides
		// changing the name in the usage text.
		res, err = internal.GofumptBytes(res)
		if err != nil {
			return err
		}
		`
	for _, path := range files {
		body := readFile(path)
		body = fixImports(body)
		name := filepath.Base(path)
		switch name {
		case "doc.go":
			continue // we have our own
		case "goimports.go":
			i := strings.Index(body, "if !bytes.Equal")
			if i < 0 {
				panic("could not insert the gofumports source code")
			}
			body = body[:i] + "\n" + extraSrc + "\n" + body[i:]
		}
		body = strings.Replace(body, "goimports", "gofumports", -1)

		writeFile(filepath.Join("gofumports", name), body)
	}
}

func copyInternal(files []string, dir string) {
	for _, path := range files {
		body := readFile(path)
		body = fixImports(body)
		name := filepath.Base(path)
		writeFile(filepath.Join(dir, name), body)
	}
}

func fixImports(body string) string {
	return strings.Replace(body,
		"golang.org/x/tools/internal/",
		"mvdan.cc/gofumpt/gofumports/internal/",
		-1)
}
