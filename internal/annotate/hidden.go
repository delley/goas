package annotate

import (
	"go/ast"
	"strings"
)

func IsHidden(astComments []*ast.Comment, showHidden bool) bool {
	for _, astComment := range astComments {
		comment := strings.TrimSpace(strings.TrimLeft(astComment.Text, "/"))
		if len(comment) == 0 {
			// ignore empty lines
			continue
		}
		attribute := strings.Fields(comment)[0]
		if strings.ToLower(attribute) == "@hidden" && !showHidden {
			return true
		}
	}
	return false
}
