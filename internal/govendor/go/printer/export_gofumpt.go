package printer

import "go/ast"

// FormatDocComment is formatDocComment, exported for gofumpt.
func FormatDocComment(list []*ast.Comment) []*ast.Comment {
	return formatDocComment(list)
}
