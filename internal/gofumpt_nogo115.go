//+build !go1.15

package internal

import (
	"go/ast"
	"path/filepath"
)

// TestscriptDirs are the directories containing test scripts.
var TestscriptDirs = []string{
	filepath.Join("testdata", "scripts"),
}

func replaceBasicLit(node *ast.BasicLit) (ast.Node, bool) {
	return node, false
}
