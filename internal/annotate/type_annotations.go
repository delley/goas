package annotate

import (
	"fmt"
	"go/ast"
	"strings"
)

func ParseApiSchemaName(commentGroup *ast.CommentGroup) (string, bool, error) {
	for _, comment := range commentGroup.List {
		fields := strings.Fields(strings.TrimLeft(comment.Text, "/"))
		if len(fields) == 0 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "@apischemaname":
			if len(fields) < 2 {
				return "", false, fmt.Errorf("expected \"// @ApiSchemaName {alias}\" received %s", comment.Text)
			}
			return fields[1], true, nil
		}
	}
	return "", false, nil
}
