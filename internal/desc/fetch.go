package desc

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
)

func FetchRef(filePath, description string) (string, error) {
	return FetchRefContext(context.Background(), filePath, description)
}

func FetchRefContext(ctx context.Context, filePath, description string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !strings.HasPrefix(description, "$ref:") {
		return description, nil
	}
	url := description[5:]
	if strings.HasPrefix(url, "file://") {
		descPath := strings.Join([]string{filePath, url[7:]}, "/")
		dat, err := os.ReadFile(descPath)
		if err != nil {
			return "", err
		}
		return string(dat), nil
	}
	// else assume http and fetch
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
