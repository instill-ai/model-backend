package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
)

type ProgressReader struct {
	r io.Reader

	filename   string
	n          float64
	lastPrintN float64
	lastPrint  time.Time
	logger     *zap.Logger
}

func NewProgressReader(r io.Reader, filename string) *ProgressReader {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := custom_logger.GetZapLogger(ctx)
	return &ProgressReader{
		r:        r,
		logger:   logger,
		filename: filename,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	bf := float64(n)
	bf /= (1 << 10)
	pr.n += bf

	if time.Since(pr.lastPrint) > time.Second ||
		(err != nil && pr.n != pr.lastPrintN) {

		pr.logger.Info(fmt.Sprintf("Copied %3.1fKiB for %s", pr.n, pr.filename))
		pr.lastPrintN = pr.n
		pr.lastPrint = time.Now()
	}
	return n, err
}

// writeToFp takes in a file pointer and byte array and writes the byte array into the file
// returns error if pointer is nil or error in writing to file
func WriteToFp(fp *os.File, data []byte) error {
	w := 0
	n := len(data)
	for {

		nw, err := fp.Write(data[w:])
		if err != nil {
			return err
		}
		w += nw
		if nw >= n {
			return nil
		}
	}
}

// GetJSON fetches the contents of the given URL
// and decodes it as JSON into the given result,
// which should be a pointer to the expected data.
func GetJSON(url string, result any) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http.Do with  MethodGet %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http.Do with  MethodGet status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	return nil
}
