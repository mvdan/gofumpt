// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"go/ast"
	"go/token"
)

func gofumpt(fset *token.FileSet, file *ast.File) {
	tfile := fset.File(file.Pos())
	cmap := ast.NewCommentMap(fset, file, file.Comments)

	removeEmpty := func(from, to token.Pos) {
		fromLine := tfile.Position(from).Line
		toLine := tfile.Position(to).Line
		for fromLine+1 < toLine {
			tfile.MergeLine(fromLine)
			toLine--
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BlockStmt:
			if len(cmap.Filter(node).Comments()) > 0 {
				// for now, skip this case.
				break
			}
			switch len(node.List) {
			case 0:
				removeEmpty(node.Lbrace, node.Rbrace)
			case 1:
				stmt := node.List[0]

				removeEmpty(node.Lbrace, stmt.Pos())
				removeEmpty(stmt.End(), node.Rbrace)
			}
		}
		return true
	})
}
