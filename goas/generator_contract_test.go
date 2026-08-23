package goas

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func exampleOptions() Options {
	return Options{
		ModulePath:   "../example",
		MainFilePath: "../example/main.go",
		OmitPackages: true,
	}
}

func TestGeneratorGenerateReturnsExampleJSON(t *testing.T) {
	generated, err := New().Generate(context.Background(), exampleOptions())
	require.NoError(t, err)

	var document interface{}
	require.NoError(t, json.Unmarshal(generated, &document))

	expected, err := os.ReadFile("../example/example.json")
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(generated))
}

func TestGeneratorGenerateToWritesJSONAndDoesNotCloseWriter(t *testing.T) {
	var output bytes.Buffer

	err := New().GenerateTo(context.Background(), exampleOptions(), &output)
	require.NoError(t, err)
	require.NotEmpty(t, output.Bytes())
	require.JSONEq(t, output.String(), string(mustReadExampleJSON(t)))

	_, err = output.WriteString(" ")
	require.NoError(t, err)
}

func TestGeneratorCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New().Generate(ctx, exampleOptions())
	require.ErrorIs(t, err, context.Canceled)

	var output bytes.Buffer
	err = New().GenerateTo(ctx, exampleOptions(), &output)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, output.Bytes())
}

func TestGeneratorGenerateToRejectsNilWriter(t *testing.T) {
	err := New().GenerateTo(context.Background(), exampleOptions(), nil)
	require.EqualError(t, err, "nil writer")
}

func mustReadExampleJSON(t *testing.T) []byte {
	t.Helper()

	expected, err := os.ReadFile("../example/example.json")
	require.NoError(t, err)
	return expected
}
