package annotate

import (
	"fmt"
	"regexp"

	"github.com/delley/goas/internal/openapi"
)

func ParseTags(comment string) (*openapi.TagDefinition, error) {
	re := regexp.MustCompile("\"([^\"]*)\"")
	matches := re.FindAllStringSubmatch(comment, -1)
	if len(matches) == 0 || len(matches[0]) == 1 {
		return nil, fmt.Errorf("expected: @Tags \"<name>\" [\"<description>\"] received: %s", comment)
	}
	tag := openapi.TagDefinition{Name: matches[0][1]}
	if len(matches) > 1 {
		tag.Description = &openapi.ReffableString{Value: matches[1][1]}
	}

	return &tag, nil
}
