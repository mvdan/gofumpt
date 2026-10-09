// Copyright (c) 2021, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package format

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/constant"
	"go/parser"
	"go/scanner"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/go-quicktest/qt"
	"golang.org/x/tools/txtar"

	"mvdan.cc/gofumpt/internal/govendor/go/format"
)

func FuzzFormat(f *testing.F) {
	// Initialize the corpus with the Go files from our test scripts.
	paths, err := filepath.Glob(filepath.Join("..", "testdata", "script", "*.txtar"))
	qt.Assert(f, qt.IsNil(err))
	qt.Assert(f, qt.Not(qt.HasLen(paths, 0)))
	for _, path := range paths {
		archive, err := txtar.ParseFile(path)
		qt.Assert(f, qt.IsNil(err))
		for _, file := range archive.Files {
			f.Logf("adding %s from %s", file.Name, path)
			if strings.HasSuffix(file.Name, ".go") || strings.Contains(file.Name, ".go.") {
				f.Add(string(file.Data), int8(18), false) // -lang=go1.18
				f.Add(string(file.Data), int8(1), false)  // -lang=go1.1
				f.Add(string(file.Data), int8(18), true)  // -lang=go1.18 -extra
			}
		}
	}

	f.Fuzz(func(t *testing.T, src string,
		majorVersion int8, // Empty version if negative, 1.N otherwise.
		extraRules bool,
	) {
		// TODO: also fuzz Options.ModulePath
		opts := Options{ExtraRules: extraRules}
		if majorVersion >= 0 {
			opts.LangVersion = fmt.Sprintf("go1.%d", majorVersion)
		}

		// go/printer counts a form feed in a comment or literal as a line
		// break, which gofumpt does not account for, as form feeds are
		// very unlikely in real code.
		src = strings.ReplaceAll(src, "\f", " ")
		orig := []byte(src)
		formatted, err := Source(orig, opts)
		if errors.As(err, &scanner.ErrorList{}) {
			return // invalid syntax from parsing
		}
		qt.Assert(t, qt.IsNil(err))

		// gofmt -s, which gofumpt's output should only differ from
		// in the ways that its rules allow.
		gofmted, err := fuzzGofmt(orig)
		qt.Assert(t, qt.IsNil(err))

		assertUnchanged := func(again []byte, err error, comment string) {
			t.Helper()
			if err != nil || !bytes.Equal(again, formatted) {
				if reason := fuzzUnsupported(orig); reason != "" {
					t.Skip(reason)
				}
				// go/printer may print invalid syntax, such as "if ({0}) {}"
				// without parens, and is not idempotent on some input,
				// such as "{ /*\n0*/ }", where each run indents the comment further.
				if regofmted, gofmtErr := fuzzGofmt(gofmted); gofmtErr != nil || !bytes.Equal(regofmted, gofmted) {
					t.Skip("gofmt -s output is invalid or unstable")
				}
			}
			qt.Assert(t, qt.IsNil(err))
			qt.Assert(t, qt.Equals(string(again), string(formatted)), qt.Commentf(comment))
		}
		again, err := Source(formatted, opts)
		assertUnchanged(again, err, "formatting is not idempotent")
		again, err = fuzzGofmt(formatted)
		assertUnchanged(again, err, "gofmt -s changes the formatted source")

		// //gofumpt:diagnose comments are rewritten to include the options.
		if opts.ExtraRules && !bytes.Contains(formatted, []byte("//gofumpt:diagnose")) {
			noExtra := opts
			noExtra.ExtraRules = false
			again, err = Source(formatted, noExtra)
			assertUnchanged(again, err, "formatting without extra rules changes the source")
		}

		// The extra rules add and remove identifiers and literals.
		qt.Assert(t, qt.DeepEquals(fuzzTokens(formatted, !opts.ExtraRules), fuzzTokens(gofmted, !opts.ExtraRules)),
			qt.Commentf("identifiers, literals, or comment words were added or removed"))

		// TODO: verify that, if the input was valid Go 1.N syntax,
		// so is the output (how? go/parser lacks an option)

		// TODO: check calling format.Node directly as well

		qt.Assert(t, qt.Equals(string(orig), src),
			qt.Commentf("input source bytes were modified"))
	})
}

// fuzzUnsupported reports why gofumpt may not format src idempotently,
// if it holds syntax which is very unlikely in real code, and which is not
// worth the complexity to support.
func fuzzUnsupported(src []byte) string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return ""
	}
	for _, group := range file.Comments {
		for _, c := range group.List {
			// go/printer moves them to the top of the file, removing them
			// along with the next line break.
			if (constraint.IsGoBuild(c.Text) || constraint.IsPlusBuild(c.Text)) && fset.Position(c.Pos()).Column > 1 {
				return "build constraint after a token or indented"
			}
			// go/doc/comment skips a run of backticks by the wrong offset,
			// turning its pairs into smart quotes a few at a time.
			// TODO: remove once internal/govendor is from Go 1.28,
			// which no longer rewrites smart quotes.
			if strings.Contains(c.Text, "```") {
				return "run of backticks in a comment"
			}
		}
	}
	reason := ""
	imports := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.EmptyStmt:
			// go/printer skips them, but the rules see them.
			if !node.Implicit {
				reason = "explicit empty statement"
			}
		case *ast.GenDecl:
			// ast.SortImports removes lines when sorting and deduplicating,
			// which joinStdImports does not always account for.
			for _, spec := range node.Specs {
				spec, ok := spec.(*ast.ImportSpec)
				if !ok {
					continue
				}
				if node.Lparen.IsValid() && fset.Position(spec.Pos()).Column == 1 {
					reason = "unindented import in a group"
				}
				if key := spec.Name.String() + " " + spec.Path.Value; imports[key] {
					reason = "duplicate import"
				} else {
					imports[key] = true
				}
			}
		}
		return reason == ""
	})
	return reason
}

// fuzzGofmt formats src like gofmt -s.
func fuzzGofmt(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	ast.SortImports(fset, file)
	simplify(file)
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fuzzTokens returns the sorted comment words in src, plus its identifiers
// and literals if code is true. Literals are kept as their constant values,
// so that e.g. 0755 equals 0o755. Punctuation in comments is ignored,
// as formatting a doc comment may rewrite list markers such as "*" to "-".
// Duplicate imports are counted once, as joining import declarations
// lets go/printer remove them. //gofumpt:diagnose comments are skipped,
// as they are rewritten to include the options.
func fuzzTokens(src []byte, code bool) []string {
	var s scanner.Scanner
	s.Init(token.NewFileSet().AddFile("", -1, len(src)), src, nil, scanner.ScanComments)
	var toks []string
	imports := make(map[string]bool)
	inImports, spec := false, ""
	for {
		_, tok, lit := s.Scan()
		if tok.IsLiteral() && tok != token.IDENT {
			lit = tok.String() + " " + constant.MakeFromLiteral(lit, tok, 0).ExactString()
		}
		switch {
		case tok == token.EOF:
			slices.Sort(toks)
			return toks
		case strings.HasPrefix(lit, "//gofumpt:diagnose"):
		case tok == token.COMMENT:
			toks = append(toks, strings.FieldsFunc(lit, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r)
			})...)
		case !code:
		case tok.IsKeyword():
			inImports = tok == token.IMPORT
		case inImports && tok == token.IDENT:
			spec = lit + " "
		case inImports && tok == token.PERIOD:
			spec = ". "
		case inImports && tok == token.STRING:
			if key := spec + lit; !imports[key] {
				imports[key] = true
				toks = append(toks, key)
			}
			spec = ""
		case tok.IsLiteral():
			toks = append(toks, lit)
		}
	}
}
