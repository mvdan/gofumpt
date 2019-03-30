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

	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BlockStmt:
			if len(node.List) != 1 {
				// we want blocks with a single statement.
				break
			}
			stmt := node.List[0]

			if len(cmap.Filter(node).Comments()) > 0 {
				// for now, skip this case.
				break
			}

			openLine := fset.Position(node.Lbrace).Line
			posLine := fset.Position(stmt.Pos()).Line
			for openLine+1 < posLine {
				tfile.MergeLine(openLine)
				posLine--
			}

			endLine := fset.Position(stmt.End()).Line
			closeLine := fset.Position(node.Rbrace).Line
			for endLine+1 < closeLine {
				tfile.MergeLine(endLine)
				closeLine--
			}
		}
		return true
	})
}
