package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"emptyclassroom/internal/config"
	cryptoutil "emptyclassroom/internal/crypto"
	"emptyclassroom/internal/model"
)

type Client struct {
	cfg   config.BUPTConfig
	http  *http.Client
	token string
}

func NewClient(cfg config.BUPTConfig) (*Client, error) {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

func (c *Client) Login(ctx context.Context) error {
	return c.login(ctx)
}

func (c *Client) FetchToday(ctx context.Context, campusID int) ([]model.ClassroomStatus, error) {
	if c.token == "" {
		if err := c.login(ctx); err != nil {
			return nil, err
		}
	}

	urlStr := c.cfg.ClassroomURL + strconv.Itoa(campusID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	req.Header.Set("token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch classroom data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch classroom data status %d: %s", resp.StatusCode, truncate(body, 512))
	}

	rooms, err := ParseClassrooms(body)
	if err != nil {
		return nil, err
	}
	for i := range rooms {
		rooms[i].CampusID = campusID
	}
	return rooms, nil
}

func (c *Client) login(ctx context.Context) error {
	pwd, err := c.passwordCiphertext()
	if err != nil {
		return err
	}
	if c.cfg.UserNo == "" {
		return fmt.Errorf("BUPT_USER_NO is required")
	}

	// Build URL with login params as query string (matches the working reference)
	parsed, _ := url.Parse(c.cfg.LoginURL)
	q := parsed.Query()
	q.Set("userNo", c.cfg.UserNo)
	q.Set("pwd", pwd)
	q.Set("encode", "1")
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("login status %d: %s", resp.StatusCode, truncate(body, 512))
	}
	if loginFailed(body) {
		return fmt.Errorf("login failed: %s", truncate(body, 512))
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Data.Token != "" {
		c.token = result.Data.Token
	}
	return nil
}

func (c *Client) passwordCiphertext() (string, error) {
	if c.cfg.EncryptedPassword != "" {
		return c.cfg.EncryptedPassword, nil
	}
	if c.cfg.Password == "" {
		return "", fmt.Errorf("BUPT_ENCRYPTED_PWD or BUPT_PASSWORD is required")
	}
	return cryptoutil.EncryptBUPTPassword(c.cfg.Password)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Referer", c.cfg.Referer)
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
}

func loginFailed(body []byte) bool {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return false
	}
	if value, ok := lookup(obj, "code"); ok {
		if v, ok := stringify(value); ok {
			return v != "1"
		}
	}
	return false
}

func lookup(obj map[string]any, key string) (any, bool) {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	for candidate, value := range obj {
		if strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(candidate)) == normalized {
			return value, true
		}
	}
	return nil, false
}

func stringify(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case float64:
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(typed, 'f', 2, 64), "0"), "."), true
	default:
		return strings.TrimSpace(fmt.Sprint(typed)), true
	}
}

func truncate(body []byte, max int) string {
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "..."
}
