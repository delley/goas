package annotate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseResponseCommentJSONTypes(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		status   string
		jsonType string
		goType   string
	}{
		{name: "object", comment: `200 object User "ok"`, status: "200", jsonType: "object", goType: "User"},
		{name: "array", comment: `200 array []User "ok"`, status: "200", jsonType: "array", goType: "[]User"},
		{name: "without payload", comment: `204 "No content"`, status: "204", jsonType: "", goType: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseResponseComment(test.comment)
			require.NoError(t, err)
			require.Equal(t, test.status, result.Status)
			require.Equal(t, test.jsonType, result.JSONType)
			require.Equal(t, test.goType, result.GoType)
		})
	}
}

func TestParseResponseCommentRejectsUnknownJSONType(t *testing.T) {
	_, err := ParseResponseComment(`200 scalar User "ok"`)
	require.EqualError(t, err, `ParseResponseComment: invalid jsonType "scalar"`)
}
