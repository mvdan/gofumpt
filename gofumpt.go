// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
)

func gofumpt(fset *token.FileSet, file *ast.File) {
	tfile := fset.File(file.Pos())
	cmap := ast.NewCommentMap(fset, file, file.Comments)

	// addNewline is a hack to let us force a newline at a certain position.
	addNewline := func(at token.Pos, plus int) {
		offset := tfile.Position(at).Offset + plus

		field := reflect.ValueOf(tfile).Elem().FieldByName("lines")
		n := field.Len()
		lines := make([]int, 0, n+1)
		for i := 0; i < n; i++ {
			prev := int(field.Index(i).Int())
			if offset >= 0 && offset < prev {
				lines = append(lines, offset)
				offset = -1
			}
			lines = append(lines, prev)
		}
		if offset >= 0 {
			lines = append(lines, offset)
		}
		if !tfile.SetLines(lines) {
			panic(fmt.Sprintf("could not set lines to %v", lines))
		}
	}

	// removeLines joins all lines between two positions, for example to
	// remove empty lines.
	removeLines := func(from, to token.Pos) {
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
				removeLines(node.Lbrace, node.Rbrace)
			case 1:
				stmt := node.List[0]

				removeLines(node.Lbrace, stmt.Pos())
				removeLines(stmt.End(), node.Rbrace)
			}
		case *ast.CompositeLit:
			if len(node.Elts) == 0 {
				break
			}
			openLine := tfile.Position(node.Lbrace).Line
			closeLine := tfile.Position(node.Rbrace).Line
			if openLine == closeLine {
				break
			}
			first := node.Elts[0]
			if openLine == tfile.Position(first.Pos()).Line {
				// We want the newline right after the brace.
				addNewline(node.Lbrace, 1)
			}
			last := node.Elts[len(node.Elts)-1]
			if closeLine == tfile.Position(last.End()).Line {
				addNewline(last.End(), 0)
			}
		}
		return true
	})
}
