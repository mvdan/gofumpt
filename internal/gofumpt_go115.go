//+build go1.15

package internal

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"regexp"
)

// TestscriptDirs are the directories containing test scripts.
var TestscriptDirs = []string{
	filepath.Join("testdata", "scripts"),
	filepath.Join("testdata", "scripts_go115"),
}

var octalIntegerLiteralRegexp = regexp.MustCompile(`\A0[0-7_]+\z`)

func replaceBasicLit(node *ast.BasicLit) (ast.Node, bool) {
	if node.Kind != token.INT || !octalIntegerLiteralRegexp.MatchString(node.Value) {
		return node, false
	}
	node.Value = "0o" + node.Value[1:]
	return node, true
}
