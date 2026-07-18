package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// restInfo is the Palworld GET /v1/api/info payload.
type restInfo struct {
	Version    string `json:"version"`
	ServerName string `json:"servername"`
	WorldGUID  string `json:"worldguid"`
}

// restMetrics is the Palworld GET /v1/api/metrics payload.
type restMetrics struct {
	CurrentPlayerNum int32 `json:"currentplayernum"`
	MaxPlayerNum     int32 `json:"maxplayernum"`
}

// RESTClient talks to the Palworld dedicated server REST API.
type RESTClient interface {
	GetInfo(ctx context.Context, baseURL, adminPassword string) (restInfo, error)
	GetMetrics(ctx context.Context, baseURL, adminPassword string) (restMetrics, error)
	// Announce broadcasts via POST /v1/api/announce {"message": "..."}.
	Announce(ctx context.Context, baseURL, adminPassword, message string) error
}

// HTTPRESTClient implements RESTClient over HTTP Basic auth (user "admin").
type HTTPRESTClient struct {
	Client *http.Client
}

func (c *HTTPRESTClient) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (c *HTTPRESTClient) GetInfo(ctx context.Context, baseURL, adminPassword string) (restInfo, error) {
	var out restInfo
	if err := c.getJSON(ctx, baseURL+"/v1/api/info", adminPassword, &out); err != nil {
		return restInfo{}, err
	}
	return out, nil
}

func (c *HTTPRESTClient) GetMetrics(ctx context.Context, baseURL, adminPassword string) (restMetrics, error) {
	var out restMetrics
	if err := c.getJSON(ctx, baseURL+"/v1/api/metrics", adminPassword, &out); err != nil {
		return restMetrics{}, err
	}
	return out, nil
}

func (c *HTTPRESTClient) Announce(ctx context.Context, baseURL, adminPassword, message string) error {
	payload, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/api/announce", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", adminPassword)
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("REST announce: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *HTTPRESTClient) getJSON(ctx context.Context, url, adminPassword string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth("admin", adminPassword)
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("REST %s: HTTP %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode REST %s: %w", url, err)
	}
	return nil
}

func restBaseURL(serviceName, namespace string, port int32) string {
	return fmt.Sprintf("http://%s.%s.svc:%d", serviceName, namespace, port)
}
