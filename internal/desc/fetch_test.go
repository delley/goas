package desc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/delley/goas/internal/openapi"
	"github.com/stretchr/testify/require"
)

func Test_fetchRef(t *testing.T) {
	t.Run("fetches local file ref", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "example.md"), []byte("Example description"), 0o600)
		require.NoError(t, err)

		desc, err := FetchRef(dir, "$ref:file://example.md")
		require.NoError(t, err)

		require.Equal(t, "Example description", desc)
	})

	t.Run("rejects paths outside the file ref path", func(t *testing.T) {
		_, err := FetchRef(t.TempDir(), "$ref:file://../example.md")
		require.Error(t, err)
		require.Contains(t, err.Error(), "escapes")
	})

	t.Run("rejects symlinks outside the file ref path", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(t.TempDir(), "outside.md")
		require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))
		require.NoError(t, os.Symlink(outside, filepath.Join(dir, "link.md")))

		_, err := FetchRef(dir, "$ref:file://link.md")
		require.Error(t, err)
		require.Contains(t, err.Error(), "escapes")
	})
}

func TestFetchRefContextReturnsErrorResponseBodyBaseline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, err := writer.Write([]byte("upstream failure"))
		require.NoError(t, err)
	}))
	defer server.Close()

	_, err := FetchRefContext(t.Context(), ".", "$ref:"+server.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "502 Bad Gateway")
	require.NotContains(t, err.Error(), "upstream failure")
}

func TestFetchRefContextHTTPStatusesAndBodyLimit(t *testing.T) {
	t.Run("accepts 2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		description, err := FetchRefContext(t.Context(), ".", "$ref:"+server.URL)
		require.NoError(t, err)
		require.Empty(t, description)
	})

	t.Run("rejects unsupported schemes", func(t *testing.T) {
		_, err := FetchRefContext(t.Context(), ".", "$ref:ftp://example.com/ref.md")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported or invalid")
	})

	t.Run("rejects an oversized body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, err := io.WriteString(writer, strings.Repeat("x", maxRefBodyBytes+1))
			require.NoError(t, err)
		}))
		defer server.Close()

		_, err := FetchRefContext(t.Context(), ".", "$ref:"+server.URL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds")
	})
}

func TestFetchRefContextPreservesContextErrors(t *testing.T) {
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := FetchRefContext(ctx, ".", "$ref:https://example.com")
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("deadline", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()
		_, err := FetchRefContext(ctx, ".", "$ref:"+server.URL)
		require.Error(t, err)
		require.True(t, errors.Is(err, context.DeadlineExceeded), err)
	})
}

func TestFetchRefContextClosesResponseBody(t *testing.T) {
	originalTransport := http.DefaultClient.Transport
	defer func() { http.DefaultClient.Transport = originalTransport }()

	for _, statusCode := range []int{http.StatusOK, http.StatusBadGateway} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			body := &trackingReadCloser{Reader: strings.NewReader("body")}
			http.DefaultClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: statusCode,
					Status:     http.StatusText(statusCode),
					Body:       body,
					Header:     make(http.Header),
					Request:    request,
				}, nil
			})

			_, _ = FetchRefContext(t.Context(), ".", "$ref:http://example.com")
			require.True(t, body.closed)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}

func Test_infoDescriptionRef(t *testing.T) {

	o := openapi.OpenAPIObject{}
	o.Info.Description = &openapi.ReffableString{Value: "$ref:http://dopeoplescroll.com/"}

	result, err := json.Marshal(o.Info.Description)

	require.NoError(t, err)
	require.Equal(t, "{\"$ref\":\"http://dopeoplescroll.com/\"}", string(result))
}
