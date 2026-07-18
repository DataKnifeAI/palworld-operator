package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TagLister lists OCI tags for a repository (e.g. ghcr.io/pocketpairjp/palserver).
type TagLister interface {
	ListTags(ctx context.Context, repository string) ([]string, error)
}

// GHCRTagLister lists tags via the OCI Distribution API with anonymous GHCR tokens.
type GHCRTagLister struct {
	Client *http.Client
	cache  *tagListCache
}

type tagListCache struct {
	mu      sync.Mutex
	entries map[string]tagCacheEntry
}

type tagCacheEntry struct {
	tags      []string
	expiresAt time.Time
}

func newTagListCache() *tagListCache {
	return &tagListCache{entries: map[string]tagCacheEntry{}}
}

func (c *tagListCache) get(key string) ([]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	out := make([]string, len(entry.tags))
	copy(out, entry.tags)
	return out, true
}

func (c *tagListCache) set(key string, tags []string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	copied := make([]string, len(tags))
	copy(copied, tags)
	c.entries[key] = tagCacheEntry{tags: copied, expiresAt: time.Now().Add(ttl)}
}

func (l *GHCRTagLister) client() *http.Client {
	if l.Client != nil {
		return l.Client
	}
	return http.DefaultClient
}

func (l *GHCRTagLister) cacheStore() *tagListCache {
	if l.cache == nil {
		l.cache = newTagListCache()
	}
	return l.cache
}

// ListTags returns tags for repository, caching successful lookups for tagCacheTTL.
func (l *GHCRTagLister) ListTags(ctx context.Context, repository string) ([]string, error) {
	host, path, err := splitOCIRepository(repository)
	if err != nil {
		return nil, err
	}
	cacheKey := host + "/" + path
	if tags, ok := l.cacheStore().get(cacheKey); ok {
		return tags, nil
	}

	token, err := l.fetchPullToken(ctx, host, path)
	if err != nil {
		return nil, err
	}

	tagsURL := fmt.Sprintf("https://%s/v2/%s/tags/list", host, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tagsURL, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := l.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("list tags %s: %w", repository, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("list tags %s: rate limited (429)", repository)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tags %s: HTTP %d: %s", repository, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode tags list: %w", err)
	}
	l.cacheStore().set(cacheKey, parsed.Tags, tagCacheTTL)
	return parsed.Tags, nil
}

func (l *GHCRTagLister) fetchPullToken(ctx context.Context, host, path string) (string, error) {
	// GHCR anonymous pull token. Other registries may not need this.
	if !strings.EqualFold(host, "ghcr.io") {
		return "", nil
	}
	tokenURL := fmt.Sprintf(
		"https://%s/token?service=%s&scope=%s",
		host,
		url.QueryEscape(host),
		url.QueryEscape("repository:"+path+":pull"),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := l.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("ghcr token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ghcr token: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode ghcr token: %w", err)
	}
	return parsed.Token, nil
}

func splitOCIRepository(repository string) (host, path string, err error) {
	repository = strings.TrimSpace(repository)
	repository = strings.TrimPrefix(repository, "https://")
	repository = strings.TrimPrefix(repository, "http://")
	repository = strings.TrimSuffix(repository, "/")
	if repository == "" {
		return "", "", fmt.Errorf("empty image repository")
	}
	// Drop accidental tag
	if slash := strings.Index(repository, "/"); slash >= 0 {
		if colon := strings.LastIndex(repository, ":"); colon > slash {
			repository = repository[:colon]
		}
	}
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid image repository %q (want registry/path)", repository)
	}
	return parts[0], parts[1], nil
}
