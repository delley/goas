package annotate

import (
	"fmt"
	"regexp"
	"strings"
)

// {name}  {in}  {goType}  {required}  {description}  		{example (optional)}
// user    body  User      true        "Info of a user."	"{\"name\":\"Bilbo\"}"
// f       file  ignored   true        "Upload a file."
var paramRe = regexp.MustCompile(`([-\w]+)[\s]+([\w]+)[\s]+([\w./\[\]\\(\\),]+)[\s]+([\w]+)[\s]+"([^"]+)"(?:[\s]+"((?:[^"\\]|\\")*)")?`)

var sliceFixRe = regexp.MustCompile(`\[\w*\]`)

func ParseParamComment(comment string) (ParamSpec, error) {
	matches := paramRe.FindStringSubmatch(comment)
	if len(matches) < 6 {
		return ParamSpec{}, fmt.Errorf("ParseParamComment can not parse param comment %q", comment)
	}

	name := matches[1]
	in := matches[2]
	goType := sliceFixRe.ReplaceAllString(matches[3], "[]")

	required := false
	switch strings.ToLower(matches[4]) {
	case "true", "required":
		required = true
	}

	desc := matches[5]
	exRaw := ""
	if len(matches) > 6 {
		exRaw = matches[6]
	}

	return ParamSpec{
		Name:        name,
		In:          in,
		GoType:      goType,
		Required:    required,
		Description: desc,
		ExampleRaw:  exRaw,
	}, nil
}
