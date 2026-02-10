package qbittorrent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client communicates with the qBittorrent Web API v2.x.
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new qBittorrent API client.
// Default timeout is 10 seconds per NFR-I2.
func NewClient(config *Config) *Client {
	jar, _ := cookiejar.New(nil)

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
			Jar:     jar,
		},
	}
}

// buildURL constructs the full API URL for the given path.
// Supports custom base paths for reverse proxy configurations (NFR-I3).
func (c *Client) buildURL(path string) string {
	host := strings.TrimSuffix(c.config.Host, "/")
	basePath := strings.TrimSuffix(c.config.BasePath, "/")
	return fmt.Sprintf("%s%s/api/v2%s", host, basePath, path)
}

// Login authenticates with qBittorrent using the configured credentials.
// On success, the session cookie is stored in the HTTP client's cookie jar.
func (c *Client) Login(ctx context.Context) error {
	loginURL := c.buildURL("/auth/login")

	data := url.Values{}
	data.Set("username", c.config.Username)
	data.Set("password", c.config.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to create login request",
			Cause:   err,
		}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "login request failed",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: fmt.Sprintf("login failed with status %d", resp.StatusCode),
		}
	}

	if string(body) != "Ok." {
		return &ConnectionError{
			Code:    ErrCodeAuthFailed,
			Message: "authentication failed: invalid credentials",
		}
	}

	slog.Info("qBittorrent login successful", "host", c.config.Host)
	return nil
}

// TestConnection verifies connectivity to qBittorrent by authenticating
// and retrieving version information.
func (c *Client) TestConnection(ctx context.Context) (*VersionInfo, error) {
	if err := c.Login(ctx); err != nil {
		return nil, err
	}

	appVersion, err := c.getVersion(ctx, "/app/version")
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to get app version",
			Cause:   err,
		}
	}

	apiVersion, err := c.getVersion(ctx, "/app/webapiVersion")
	if err != nil {
		return nil, &ConnectionError{
			Code:    ErrCodeConnectionFailed,
			Message: "failed to get API version",
			Cause:   err,
		}
	}

	return &VersionInfo{
		AppVersion: appVersion,
		APIVersion: apiVersion,
	}, nil
}

// getVersion fetches a version string from the given API path.
func (c *Client) getVersion(ctx context.Context, path string) (string, error) {
	versionURL := c.buildURL(path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(body)), nil
}
