package annotate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// {status}  {jsonType}  {goType}     {description}
// 201       object      models.User  "User Model"
// if 204 or something else without empty return payload
// 204 "User Model"
var responseRe = regexp.MustCompile(`(?P<status>[\d]+)[\s]*(?P<jsonType>[\w\{\}]+)?[\s]+(?P<goType>[\w\-\.\/\[\]]+)?[^"]*(?P<description>.*)?`)

var sliceFixRespRe = regexp.MustCompile(`\[\w*\]`)

func ParseResponseComment(comment string) (ResponseSpec, error) {
	matches := responseRe.FindStringSubmatch(comment)

	paramsMap := make(map[string]string)
	for i, name := range responseRe.SubexpNames() {
		if i > 0 && i <= len(matches) {
			paramsMap[name] = matches[i]
		}
	}

	if len(matches) <= 2 {
		return ResponseSpec{}, fmt.Errorf("ParseResponseComment can not parse response comment %q, matches: %v", comment, matches)
	}

	status := paramsMap["status"]
	if _, err := strconv.Atoi(status); err != nil {
		return ResponseSpec{}, fmt.Errorf("ParseResponseComment: http status must be int, but got %s", status)
	}

	// ignore type if not set
	if jsonType := paramsMap["jsonType"]; jsonType != "" {
		switch jsonType {
		case "object", "array", "{object}", "{array}":
		default:
			return ResponseSpec{}, fmt.Errorf("ParseResponseComment: invalid jsonType %q", jsonType)
		}
	}

	desc := strings.Trim(paramsMap["description"], "\"")

	goType := ""
	if raw := paramsMap["goType"]; raw != "" {
		goType = sliceFixRespRe.ReplaceAllString(raw, "[]")
	}

	return ResponseSpec{
		Status:      status,
		JSONType:    paramsMap["jsonType"],
		GoType:      goType,
		Description: desc,
	}, nil
}
