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
			if len(cmap.Filter(node).Comments()) > 0 {
				// for now, skip this case.
				break
			}
			openLine := fset.Position(node.Lbrace).Line
			closeLine := fset.Position(node.Rbrace).Line

			if len(node.List) == 0 {
				for openLine+1 < closeLine {
					tfile.MergeLine(openLine)
					closeLine--
				}
				break
			}

			if len(node.List) != 1 {
				// we want blocks with a single statement.
				break
			}
			stmt := node.List[0]

			posLine := fset.Position(stmt.Pos()).Line
			for openLine+1 < posLine {
				tfile.MergeLine(openLine)
				posLine--
				closeLine--
			}

			endLine := fset.Position(stmt.End()).Line
			for endLine+1 < closeLine {
				tfile.MergeLine(endLine)
				closeLine--
			}
		}
		return true
	})
}
