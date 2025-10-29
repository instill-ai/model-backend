package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"

	logx "github.com/instill-ai/x/log"
)

const (
	reqTimeout    = time.Second * 30
	maxRetryCount = 3
	retryDelay    = 100 * time.Millisecond
)

// RegistryClient interacts with the Docker Registry HTTP V2 API.
type RegistryClient struct {
	*resty.Client
}

// NewRegistryClient returns an initialized registry HTTP client.
func NewRegistryClient(ctx context.Context, registryHost string, registryPort int) *RegistryClient {
	logger, _ := logx.GetZapLogger(ctx)
	baseURL := fmt.Sprintf("http://%s:%d", registryHost, registryPort)

	r := resty.New().
		SetLogger(logger.Sugar()).
		SetBaseURL(baseURL).
		SetTimeout(reqTimeout).
		SetTransport(&http.Transport{
			DisableKeepAlives: true,
		}).
		SetRetryCount(maxRetryCount).
		SetRetryWaitTime(retryDelay)

	return &RegistryClient{Client: r}
}

type tagList struct {
	Tags []string `json:"tags"`
}

// ListTags calls the GET /v2/<name>/tags/list endpoint, where <name> is a
// repository.
func (c *RegistryClient) ListTags(ctx context.Context, repository string) ([]string, error) {
	var resp tagList

	tagsPath := fmt.Sprintf("/v2/%s/tags/list", repository)
	r := c.R().SetContext(ctx).SetResult(&resp)
	if _, err := r.Get(tagsPath); err != nil {
		return nil, fmt.Errorf("couldn't connect with registry: %w", err)
	}

	return resp.Tags, nil
}

// DeleteTag calls the DELETE /v2/<name>/manifests/<reference> endpoint, where <name> is a
// repository, and <reference> is the digest
func (c *RegistryClient) DeleteTag(ctx context.Context, repository string, digest string) error {

	deletePath := fmt.Sprintf("/v2/%s/manifests/%s", repository, digest)
	r := c.R().SetContext(ctx)
	if _, err := r.Delete(deletePath); err != nil {
		return fmt.Errorf("couldn't delete the image with registry: %w", err)
	}

	return nil
}

// GetTagDigest calls the HEAD /v2/<name>/manifests/<reference> endpoint, where <name> is a
// repository, and <reference> is the tag
func (c *RegistryClient) GetTagDigest(ctx context.Context, repository string, tag string) (string, error) {

	digestPath := fmt.Sprintf("/v2/%s/manifests/%s", repository, tag)
	r := c.R().SetContext(ctx).SetHeader("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := r.Head(digestPath)
	if err != nil {
		return "", fmt.Errorf("couldn't get the image digest: %w", err)
	}

	return resp.Header().Get("Docker-Content-Digest"), nil
}
