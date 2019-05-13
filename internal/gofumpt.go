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

	"golang.org/x/tools/go/ast/astutil"
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
	pre := func(c *astutil.Cursor) bool {
		f.applyPre(c)
		if _, ok := c.Node().(*ast.BlockStmt); ok {
			f.blockLevel++
		}
		return true
	}
	post := func(c *astutil.Cursor) bool {
		if _, ok := c.Node().(*ast.BlockStmt); ok {
			f.blockLevel--
		}
		return true
	}
	astutil.Apply(file, pre, post)
}

type fumpter struct {
	fset *token.FileSet
	file *token.File

	astFile *ast.File

	blockLevel int
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
	return int(count) + (f.blockLevel * 8)
}

// rxCommentDirective covers all common Go comment directives:
//
//   //go:        | standard Go directives, like go:noinline
//   //someword:  | similar to the syntax above, like lint:ignore
//   //line       | inserted line information for cmd/compile
//   //export     | to mark cgo funcs for exporting
//   //extern     | C function declarations for gccgo
//   //sys(nb)?   | syscall function wrapper prototypes
var rxCommentDirective = regexp.MustCompile(`^([a-z]+:|line\b|export\b|extern\b|sys(nb)?\b)`)

// visit takes either an ast.Node or a []ast.Stmt.
func (f *fumpter) applyPre(c *astutil.Cursor) {
	switch node := c.Node().(type) {
	case *ast.File:
		var lastMulti bool
		var lastEnd token.Pos
		for _, decl := range node.Decls {
			pos := decl.Pos()
			comments := f.commentsBetween(lastEnd, pos)
			if len(comments) > 0 {
				pos = comments[0].Pos()
			}

			// multiline top-level declarations should be separated
			multi := f.posLine(decl.Pos()) < f.posLine(decl.End())
			if (multi && lastMulti) &&
				f.posLine(lastEnd)+1 == f.posLine(pos) {
				f.addNewline(lastEnd)
			}

			lastMulti = multi
			lastEnd = decl.End()
		}

		// Comments aren't nodes, so they're not walked by default.
		for _, group := range node.Comments {
			for _, comment := range group.List {
				body := strings.TrimPrefix(comment.Text, "//")
				if body == comment.Text {
					// /*-style comment
					break
				}
				if rxCommentDirective.MatchString(body) {
					// this comment is a directive
					break
				}
				r, _ := utf8.DecodeRuneInString(body)
				if unicode.IsLetter(r) || unicode.IsNumber(r) {
					comment.Text = "// " + body
				}
			}
		}

	case *ast.DeclStmt:
		decl, ok := node.Decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR || len(decl.Specs) != 1 {
			break // e.g. const name = "value"
		}
		spec := decl.Specs[0].(*ast.ValueSpec)
		if spec.Type != nil {
			break // e.g. var name Type
		}
		names := make([]ast.Expr, len(spec.Names))
		for i, name := range spec.Names {
			names[i] = name
		}
		c.Replace(&ast.AssignStmt{
			Lhs: names,
			Tok: token.DEFINE,
			Rhs: spec.Values,
		})

	case *ast.GenDecl:
		if node.Tok == token.IMPORT && node.Lparen.IsValid() {
			f.joinStdImports(node)
		}
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
		f.stmts(node.List)
		comments := f.commentsBetween(node.Lbrace, node.Rbrace)
		if len(node.List) == 0 && len(comments) == 0 {
			f.removeLinesBetween(node.Lbrace, node.Rbrace)
			break
		}

		isFuncBody := false
		switch c.Parent().(type) {
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
		f.stmts(node.Body)
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

	case *ast.CommClause:
		f.stmts(node.Body)
	}
}

func (f *fumpter) stmts(list []ast.Stmt) {
	for i, stmt := range list {
		ifs, ok := stmt.(*ast.IfStmt)
		if !ok || i < 1 {
			continue // not an if following another statement
		}
		as, ok := list[i-1].(*ast.AssignStmt)
		if !ok || as.Tok != token.DEFINE ||
			!identEqual(as.Lhs[len(as.Lhs)-1], "err") {
			continue // not "..., err := ..."
		}
		be, ok := ifs.Cond.(*ast.BinaryExpr)
		if !ok || ifs.Init != nil || ifs.Else != nil {
			continue // complex if
		}
		if be.Op != token.NEQ || !identEqual(be.X, "err") ||
			!identEqual(be.Y, "nil") {
			continue // not "err != nil"
		}
		f.removeLinesBetween(as.End(), ifs.Pos())
	}
}

func identEqual(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// joinStdImports ensures that all standard library imports are together and at
// the top of the imports list.
func (f *fumpter) joinStdImports(d *ast.GenDecl) {
	var std, other []ast.Spec
	for _, spec := range d.Specs {
		spec := spec.(*ast.ImportSpec)
		// First, separate the non-std imports.
		if strings.Contains(spec.Path.Value, ".") {
			other = append(other, spec)
			continue
		}
		if len(other) > 0 {
			// If we're moving this std import further up, reset its
			// position, to avoid breaking comments.
			setPos(reflect.ValueOf(spec), d.Pos())
		}
		std = append(std, spec)
	}
	// Finally, join the imports, keeping std at the top.
	d.Specs = append(std, other...)
}

var posType = reflect.TypeOf(token.NoPos)

// setPos recursively sets all position fields in the node v to pos.
func setPos(v reflect.Value, pos token.Pos) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		return
	}
	if v.Type() == posType {
		v.Set(reflect.ValueOf(pos))
	}
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			setPos(v.Field(i), pos)
		}
	}
}
