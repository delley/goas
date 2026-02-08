package desc

import (
	"io"
	"net/http"
	"os"
	"strings"
)

func FetchRef(filePath, description string) (string, error) {
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
	resp, err := http.Get(url)
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
