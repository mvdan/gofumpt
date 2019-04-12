// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Multiline nodes which could fit on a single line under this many
// bytes may be collapsed onto a single line.
const shortLineLimit = 60

func GofumptBytes(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	Gofumpt(fset, file)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Gofumpt(fset *token.FileSet, file *ast.File) {
	f := &fumpter{
		fset:    fset,
		file:    fset.File(file.Pos()),
		astFile: file,
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			f.stack = f.stack[:len(f.stack)-1]
			return true
		}
		f.visit(node)
		f.stack = append(f.stack, node)
		return true
	})
}

type fumpter struct {
	fset *token.FileSet
	file *token.File

	astFile *ast.File

	stack []ast.Node
}

func (f *fumpter) posLine(pos token.Pos) int {
	return f.file.Position(pos).Line
}

func (f *fumpter) commentsBetween(p1, p2 token.Pos) []*ast.CommentGroup {
	comments := f.astFile.Comments
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

func (f *fumpter) inlineComment(pos token.Pos) *ast.Comment {
	comments := f.astFile.Comments
	i := sort.Search(len(comments), func(i int) bool {
		return comments[i].Pos() >= pos
	})
	if i >= len(comments) {
		return nil
	}
	line := f.posLine(pos)
	for _, comment := range comments[i].List {
		if f.posLine(comment.Pos()) == line {
			return comment
		}
	}
	return nil
}

// addNewline is a hack to let us force a newline at a certain position.
func (f *fumpter) addNewline(at token.Pos) {
	offset := f.file.Position(at).Offset

	field := reflect.ValueOf(f.file).Elem().FieldByName("lines")
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
	if !f.file.SetLines(lines) {
		panic(fmt.Sprintf("could not set lines to %v", lines))
	}
}

// removeNewlines removes all newlines between two positions, so that they end
// up on the same line.
func (f *fumpter) removeLines(fromLine, toLine int) {
	for fromLine < toLine {
		f.file.MergeLine(fromLine)
		toLine--
	}
}

// removeLinesBetween is like removeLines, but it leaves one newline between the
// two positions.
func (f *fumpter) removeLinesBetween(from, to token.Pos) {
	f.removeLines(f.posLine(from)+1, f.posLine(to))
}

type byteCounter int

func (b *byteCounter) Write(p []byte) (n int, err error) {
	*b += byteCounter(len(p))
	return len(p), nil
}

func (f *fumpter) printLength(node ast.Node) int {
	var count byteCounter
	if err := format.Node(&count, f.fset, node); err != nil {
		panic(fmt.Sprintf("unexpected print error: %v", err))
	}

	// Add the space taken by an inline comment.
	if c := f.inlineComment(node.End()); c != nil {
		fmt.Fprintf(&count, " %s", c.Text)
	}

	// Add an approximation of the indentation level. We can't know the
	// number of tabs go/printer will add ahead of time. Trying to print the
	// entire top-level declaration would tell us that, but then it's near
	// impossible to reliably find our node again.
	for _, parent := range f.stack {
		if _, ok := parent.(*ast.BlockStmt); ok {
			count += 8
		}
	}
	return int(count)
}

// rxCommentDirective covers all common Go comment directives:
//
//   //go:        | standard Go directives, like go:noinline
//   //someword:  | similar to the syntax above, like lint:ignore
//   //line       | inserted line information for cmd/compile
//   //export     | to mark cgo funcs for exporting
var rxCommentDirective = regexp.MustCompile(`^([a-z]+:|line\b|export\b)`)

func (f *fumpter) visit(node ast.Node) {
	switch node := node.(type) {
	case *ast.File:
		var lastMulti bool
		var lastEnd token.Pos
		for _, decl := range node.Decls {
			pos := decl.Pos()
			comments := f.commentsBetween(lastEnd, pos)
			if len(comments) > 0 {
				pos = comments[0].Pos()
			}

			multi := f.posLine(decl.Pos()) < f.posLine(decl.End())
			if (multi && lastMulti) &&
				f.posLine(lastEnd)+1 == f.posLine(pos) {
				f.addNewline(lastEnd)
			}

			lastMulti = multi
			lastEnd = decl.End()
		}

		// The unattached comments are ignored by ast.Walk.
		// Don't bother with the stack.
		for _, group := range node.Comments {
			f.visit(group)
			for _, comment := range group.List {
				f.visit(comment)
			}
		}

	case *ast.Comment:
		body := strings.TrimPrefix(node.Text, "//")
		if body == node.Text {
			// /*-style comment
			break
		}
		if rxCommentDirective.MatchString(body) {
			// this comment is a directive
			break
		}
		r, _ := utf8.DecodeRuneInString(body)
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			node.Text = "// " + body
		}

	case *ast.GenDecl:
		if len(node.Specs) == 1 && node.Lparen.IsValid() {
			// If the single spec has any comment, it must go before
			// the entire declaration now.
			node.TokPos = node.Specs[0].Pos()

			// Remove the parentheses. go/printer will automatically
			// get rid of the newlines.
			node.Lparen = token.NoPos
			node.Rparen = token.NoPos
		}

	case *ast.BlockStmt:
		comments := f.commentsBetween(node.Lbrace, node.Rbrace)
		if len(node.List) == 0 && len(comments) == 0 {
			f.removeLinesBetween(node.Lbrace, node.Rbrace)
			break
		}

		isFuncBody := false
		switch f.stack[len(f.stack)-1].(type) {
		case *ast.FuncDecl:
			isFuncBody = true
		case *ast.FuncLit:
			isFuncBody = true
		}

		if len(node.List) > 1 && !isFuncBody {
			// only if we have a single statement, or if
			// it's a func body.
			break
		}
		var bodyPos, bodyEnd token.Pos

		if len(node.List) > 0 {
			bodyPos = node.List[0].Pos()
			bodyEnd = node.List[len(node.List)-1].End()
		}
		if len(comments) > 0 {
			if pos := comments[0].Pos(); !bodyPos.IsValid() || pos < bodyPos {
				bodyPos = pos
			}
			if pos := comments[len(comments)-1].End(); !bodyPos.IsValid() || pos > bodyEnd {
				bodyEnd = pos
			}
		}

		f.removeLinesBetween(node.Lbrace, bodyPos)
		f.removeLinesBetween(bodyEnd, node.Rbrace)

	case *ast.CompositeLit:
		if len(node.Elts) == 0 {
			// doesn't have elements
			break
		}
		openLine := f.posLine(node.Lbrace)
		closeLine := f.posLine(node.Rbrace)
		if openLine == closeLine {
			// all in a single line
			break
		}

		newlineBetweenElems := false
		lastLine := openLine
		for _, elem := range node.Elts {
			if f.posLine(elem.Pos()) > lastLine {
				newlineBetweenElems = true
			}
			lastLine = f.posLine(elem.End())
		}
		if closeLine > lastLine {
			newlineBetweenElems = true
		}

		if !newlineBetweenElems {
			// no newlines between elements (and braces)
			break
		}

		first := node.Elts[0]
		if openLine == f.posLine(first.Pos()) {
			// We want the newline right after the brace.
			f.addNewline(node.Lbrace + 1)
			closeLine = f.posLine(node.Rbrace)
		}
		last := node.Elts[len(node.Elts)-1]
		if closeLine == f.posLine(last.End()) {
			f.addNewline(last.End())
		}

	case *ast.CaseClause:
		openLine := f.posLine(node.Case)
		closeLine := f.posLine(node.Colon)
		if openLine == closeLine {
			// nothing to do
			break
		}
		if len(f.commentsBetween(node.Case, node.Colon)) > 0 {
			// don't move comments
			break
		}
		if f.printLength(node) > shortLineLimit {
			// too long to collapse
			break
		}
		f.removeLines(openLine, closeLine)
	}
}
