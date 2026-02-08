package annotate

import (
	"encoding/json"
	"strings"
)

func ParseRequestBodyExample(example string) (interface{}, error) {
	var exampleRequestBody interface{}
	err := json.Unmarshal([]byte(strings.Replace(example, "\\\"", "\"", -1)), &exampleRequestBody)
	if err != nil {
		return nil, err
	}
	return exampleRequestBody, nil
}
