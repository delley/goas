package desc

import (
	"encoding/json"
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/stretchr/testify/require"
)

func Test_fetchRef(t *testing.T) {
	t.Run("fetches local file ref", func(t *testing.T) {
		desc, err := FetchRef(".", "$ref:file://../../example/example.md")
		require.NoError(t, err)

		require.Equal(t, "Example description", desc)
	})
}

func Test_infoDescriptionRef(t *testing.T) {

	o := openapi.OpenAPIObject{}
	o.Info.Description = &openapi.ReffableString{Value: "$ref:http://dopeoplescroll.com/"}

	result, err := json.Marshal(o.Info.Description)

	require.NoError(t, err)
	require.Equal(t, "{\"$ref\":\"http://dopeoplescroll.com/\"}", string(result))
}
