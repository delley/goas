package load

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SortedKeys(t *testing.T) {
	t.Run("returns sorted keys", func(t *testing.T) {
		m := map[string]int{
			"b": 2,
			"a": 1,
			"c": 3,
		}

		keys := SortedKeys(m)

		require.Equal(t, []string{"a", "b", "c"}, keys)
	})

	t.Run("empty map", func(t *testing.T) {
		m := map[string]struct{}{}

		keys := SortedKeys(m)

		require.Empty(t, keys)
	})

	t.Run("nil map", func(t *testing.T) {
		var m map[string]any

		keys := SortedKeys(m)

		require.Empty(t, keys)
	})
}
