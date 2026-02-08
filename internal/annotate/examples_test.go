package annotate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseRequestBodyExample(t *testing.T) {
	t.Run("Parses example object request body", func(t *testing.T) {
		exampleRequestBody, err := ParseRequestBodyExample("{\\\"name\\\":\\\"Bilbo\\\"}")
		require.NoError(t, err)

		require.Equal(t, map[string]interface{}(map[string]interface{}{"name": "Bilbo"}), exampleRequestBody)
	})

	t.Run("Parses example array request body", func(t *testing.T) {
		exampleRequestBody, err := ParseRequestBodyExample("[{\\\"name\\\":\\\"Bilbo\\\"}]")
		require.NoError(t, err)

		require.Equal(t, []interface{}([]interface{}{map[string]interface{}{"name": "Bilbo"}}), exampleRequestBody)
	})

	t.Run("Errors if example is invalid", func(t *testing.T) {
		_, err := ParseRequestBodyExample("{name:\\\"Smaug\\\"}")
		require.Error(t, err)
	})
}
