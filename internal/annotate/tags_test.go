package annotate

import (
	"encoding/json"
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/stretchr/testify/require"
)

func Test_parseTags(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		result, err := ParseTags("@Tags \"Foo\"")

		require.NoError(t, err)
		require.Equal(t, &openapi.TagDefinition{Name: "Foo"}, result)
	})

	t.Run("name and description", func(t *testing.T) {
		result, err := ParseTags("@Tags \"Foobar\" \"Barbaz\"")

		require.NoError(t, err)
		require.Equal(t, &openapi.TagDefinition{Name: "Foobar", Description: &openapi.ReffableString{Value: "Barbaz"}}, result)
	})

	t.Run("name and description including ref ", func(t *testing.T) {
		result, err := ParseTags("@Tags \"Foobar\" \"$ref:path/to/baz\"")
		require.NoError(t, err)
		b, err := json.Marshal(result)
		require.NoError(t, err)
		require.Equal(t, "{\"name\":\"Foobar\",\"description\":{\"$ref\":\"path/to/baz\"}}", string(b))
	})

	t.Run("invalid tag", func(t *testing.T) {
		_, err := ParseTags("@Tags Foobar Barbaz")

		require.Error(t, err)
	})
}
