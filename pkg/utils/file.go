package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// GetJSON fetches the contents from the given URL or reads from a local file path
// and decodes it as JSON into the given result,
// which should be a pointer to the expected data.
func GetJSON(urlOrPath string, result any) error {
	// Check if the input is a URL or local file path
	if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
		// Handle as URL
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlOrPath, http.NoBody)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("http.Do with MethodGet %q: %w", urlOrPath, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("http.Do with MethodGet status: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("io.ReadAll: %w", err)
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("json.Unmarshal: %w", err)
		}
	} else {
		// Handle as local file path
		data, err := os.ReadFile(urlOrPath)
		if err != nil {
			return fmt.Errorf("os.ReadFile %q: %w", urlOrPath, err)
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("json.Unmarshal: %w", err)
		}
	}

	return nil
}
