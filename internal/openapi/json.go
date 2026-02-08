package openapi

import (
	"encoding/json"
	"errors"
	"strings"
)

func (r ReffableString) MarshalJSON() ([]byte, error) {
	if strings.HasPrefix(r.Value, "$ref:") {
		if r.Value == "$ref:" {
			return nil, errors.New("$ref is missing URL")
		}
		// encode as a reference object instead of a string
		return json.Marshal(reference{Ref: r.Value[5:]})
	}
	return json.Marshal(r.Value)
}
