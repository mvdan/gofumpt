// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"reflect"
	"strings"

	"mvdan.cc/gofumpt/internal/util"
)

func initRewrite() {
	if *rewriteRule == "" {
		rewrite = nil // disable any previous rewrite
		return
	}
	f := strings.Split(*rewriteRule, "->")
	if len(f) != 2 {
		fmt.Fprintf(os.Stderr, "rewrite rule must be of the form 'pattern -> replacement'\n")
		os.Exit(2)
	}
	pattern := parseExpr(f[0], "pattern")
	replace := parseExpr(f[1], "replacement")
	rewrite = func(p *ast.File) *ast.File { return rewriteFile(pattern, replace, p) }
}

// parseExpr parses s as an expression.
// It might make sense to expand this to allow statement patterns,
// but there are problems with preserving formatting and also
// with what a wildcard for a statement looks like.
func parseExpr(s, what string) ast.Expr {
	x, err := parser.ParseExpr(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing %s %s at %s\n", what, s, err)
		os.Exit(2)
	}
	return x
}

// Keep this function for debugging.
/*
func dump(msg string, val reflect.Value) {
	fmt.Printf("%s:\n", msg)
	ast.Print(fileSet, val.Interface())
	fmt.Println()
}
*/

// rewriteFile applies the rewrite rule 'pattern -> replace' to an entire file.
func rewriteFile(pattern, replace ast.Expr, p *ast.File) *ast.File {
	cmap := ast.NewCommentMap(fileSet, p, p.Comments)
	m := make(map[string]reflect.Value)
	pat := reflect.ValueOf(pattern)
	repl := reflect.ValueOf(replace)

	var rewriteVal func(val reflect.Value) reflect.Value
	rewriteVal = func(val reflect.Value) reflect.Value {
		// don't bother if val is invalid to start with
		if !val.IsValid() {
			return reflect.Value{}
		}
		val = util.Apply(rewriteVal, val)
		for k := range m {
			delete(m, k)
		}
		if util.Match(m, pat, val) {
			val = util.Subst(m, repl, reflect.ValueOf(val.Interface().(ast.Node).Pos()))
		}
		return val
	}

	r := util.Apply(rewriteVal, reflect.ValueOf(p)).Interface().(*ast.File)
	r.Comments = cmap.Filter(r).Comments() // recreate comments list
	return r
}
