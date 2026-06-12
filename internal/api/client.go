package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sim4gh/mkpdfs-cli/internal/auth"
	"github.com/sim4gh/mkpdfs-cli/internal/config"
	"github.com/sim4gh/mkpdfs-cli/internal/envs"
)

type Response struct {
	StatusCode int
	Body       json.RawMessage
}

func (r *Response) Unmarshal(v any) error { return json.Unmarshal(r.Body, v) }

type Client struct {
	BaseURL string
	Env     envs.Env
	token   string // bearer, set lazily
	apiKey  string
}

var httpClient = &http.Client{Timeout: 120 * time.Second} // PDF generation can take a while

// New builds a client for the env, without auth resolved yet.
func New(env envs.Env) *Client {
	return &Client{BaseURL: env.APIBase, Env: env}
}

// WithJWT ensures a valid (refreshed) JWT and attaches it.
func (c *Client) WithJWT() (*Client, error) {
	creds := config.Get().Creds(c.Env.Name)
	if creds.IDToken == "" {
		return nil, errors.New(`not authenticated. Run "mkp auth login"`)
	}
	if auth.IsTokenExpired(creds.IDToken) {
		refreshed, err := auth.RefreshTokens(c.Env)
		if err != nil {
			return nil, err
		}
		creds = refreshed
	}
	c.token = creds.IDToken
	return c, nil
}

// WithAPIKey attaches the API key; MKPDFS_API_KEY env var wins over config.
func (c *Client) WithAPIKey() (*Client, error) {
	key := os.Getenv("MKPDFS_API_KEY")
	if key == "" {
		key = config.Get().Creds(c.Env.Name).APIKey
	}
	if key == "" {
		return nil, errors.New(`no API key configured. Run "mkp tokens create --save" or set MKPDFS_API_KEY`)
	}
	c.apiKey = key
	return c, nil
}

func (c *Client) do(method, path string, body any, headers map[string]string) (*Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mkp")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to reach %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return &Response{StatusCode: resp.StatusCode, Body: raw}, parseAPIError(resp.StatusCode, raw)
	}
	return &Response{StatusCode: resp.StatusCode, Body: raw}, nil
}

func (c *Client) Get(path string) (*Response, error)            { return c.do("GET", path, nil, nil) }
func (c *Client) Post(path string, body any) (*Response, error) { return c.do("POST", path, body, nil) }
func (c *Client) Put(path string, body any) (*Response, error)  { return c.do("PUT", path, body, nil) }
func (c *Client) Delete(path string) (*Response, error)         { return c.do("DELETE", path, nil, nil) }

func (c *Client) PostWithKey(path string, body any) (*Response, error) {
	return c.do("POST", path, body, map[string]string{"x-api-key": c.apiKey})
}

// parseAPIError prefers the structured body (backend status codes are
// inconsistent — limits surface as 403 or 429 depending on the path).
func parseAPIError(status int, body []byte) error {
	var m struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(body, &m) == nil {
		if m.Message != "" {
			return errors.New(m.Message)
		}
		if m.Error != "" {
			return errors.New(m.Error)
		}
	}
	return fmt.Errorf("API error (status %d)", status)
}
