// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"sort"
)

func gofumpt(fset *token.FileSet, file *ast.File) {
	tfile := fset.File(file.Pos())

	posLine := func(pos token.Pos) int { return tfile.Position(pos).Line }

	commentsBetween := func(p1, p2 token.Pos) []*ast.CommentGroup {
		comments := file.Comments
		i1 := sort.Search(len(comments), func(i int) bool {
			return comments[i].Pos() >= p1
		})
		comments = comments[i1:]
		i2 := sort.Search(len(comments), func(i int) bool {
			return comments[i].Pos() >= p2
		})
		comments = comments[:i2]
		return comments
	}

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
		fromLine := posLine(from)
		toLine := posLine(to)
		for fromLine+1 < toLine {
			tfile.MergeLine(fromLine)
			toLine--
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BlockStmt:
			comments := commentsBetween(node.Lbrace, node.Rbrace)
			if len(comments) > 0 {
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
				// doesn't have elements
				break
			}
			openLine := posLine(node.Lbrace)
			closeLine := posLine(node.Rbrace)
			if openLine == closeLine {
				// all in a single line
				break
			}

			newlineBetweenElems := false
			lastLine := openLine
			for _, elem := range node.Elts {
				if posLine(elem.Pos()) > lastLine {
					newlineBetweenElems = true
				}
				lastLine = posLine(elem.End())
			}
			if closeLine > lastLine {
				newlineBetweenElems = true
			}

			if !newlineBetweenElems {
				// no newlines between elements (and braces)
				break
			}

			first := node.Elts[0]
			if openLine == posLine(first.Pos()) {
				// We want the newline right after the brace.
				addNewline(node.Lbrace, 1)
				closeLine = posLine(node.Rbrace)
			}
			last := node.Elts[len(node.Elts)-1]
			if closeLine == posLine(last.End()) {
				addNewline(last.End(), 0)
			}
		}
		return true
	})
}
