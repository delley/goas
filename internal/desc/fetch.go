package desc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func FetchRef(filePath, description string) (string, error) {
	return FetchRefContext(context.Background(), filePath, description)
}

const (
	defaultRefTimeout = 30 * time.Second
	maxRefBodyBytes   = 1 << 20
)

func FetchRefContext(ctx context.Context, filePath, description string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !strings.HasPrefix(description, "$ref:") {
		return description, nil
	}
	refURL := description[5:]
	if strings.HasPrefix(refURL, "file://") {
		descPath, err := resolveFileRef(filePath, refURL[7:])
		if err != nil {
			return "", err
		}
		dat, err := os.ReadFile(descPath)
		if err != nil {
			return "", fmt.Errorf("read file ref %q: %w", descPath, err)
		}
		return string(dat), nil
	}

	parsedURL, err := url.Parse(refURL)
	if err != nil || (strings.ToLower(parsedURL.Scheme) != "http" && strings.ToLower(parsedURL.Scheme) != "https") || parsedURL.Host == "" {
		return "", fmt.Errorf("unsupported or invalid reference URL %q", refURL)
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRefTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, refURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch ref %q: %w", refURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("fetch ref %q: unexpected HTTP status %s", refURL, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRefBodyBytes+1))
	if err != nil {
		return "", fmt.Errorf("read ref %q: %w", refURL, err)
	}
	if len(body) > maxRefBodyBytes {
		return "", fmt.Errorf("read ref %q: response body exceeds %d bytes", refURL, maxRefBodyBytes)
	}

	return string(body), nil
}

func resolveFileRef(filePath, refPath string) (string, error) {
	if refPath == "" || filepath.IsAbs(refPath) {
		return "", fmt.Errorf("file ref path %q must be relative", refPath)
	}
	basePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("resolve file ref base %q: %w", filePath, err)
	}
	basePath, err = filepath.EvalSymlinks(basePath)
	if err != nil {
		return "", fmt.Errorf("resolve file ref base %q: %w", filePath, err)
	}
	descPath := filepath.Join(basePath, filepath.Clean(refPath))
	descPath, err = filepath.Abs(descPath)
	if err != nil {
		return "", fmt.Errorf("resolve file ref %q: %w", refPath, err)
	}
	relPath, err := filepath.Rel(basePath, descPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file ref path %q escapes %q", refPath, filePath)
	}
	resolvedPath, err := filepath.EvalSymlinks(descPath)
	if err != nil {
		return descPath, nil
	}
	relPath, err = filepath.Rel(basePath, resolvedPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file ref path %q escapes %q", refPath, filePath)
	}
	return descPath, nil
}
