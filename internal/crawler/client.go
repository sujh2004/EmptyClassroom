package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
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
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Jar:     jar,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.classroomURL(campusID), nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

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

	form := url.Values{}
	form.Set("userNo", c.cfg.UserNo)
	form.Set("pwd", pwd)
	form.Set("encode", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.LoginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

func (c *Client) classroomURL(campusID int) string {
	parsed, err := url.Parse(c.cfg.ClassroomURL)
	if err != nil {
		return c.cfg.ClassroomURL
	}
	query := parsed.Query()
	query.Set("campusId", strconv.Itoa(campusID))
	if c.token != "" {
		query.Set("token", c.token)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
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
	if value, ok := lookup(obj, "success"); ok {
		if success, ok := value.(bool); ok {
			return !success
		}
	}
	if value, ok := lookup(obj, "code"); ok {
		msg := ""
		if m, ok := lookup(obj, "message"); ok {
			msg = strings.ToLower(stringify(m))
		} else if m, ok := lookup(obj, "msg"); ok {
			msg = strings.ToLower(stringify(m))
		}
		if msg != "" {
			if strings.Contains(msg, "fail") || strings.Contains(msg, "error") || strings.Contains(msg, "错误") || strings.Contains(msg, "失败") || strings.Contains(msg, "不存在") || strings.Contains(msg, "非法") {
				switch code := value.(type) {
				case float64:
					return code != 1 && code != 200
				case string:
					return code != "1" && code != "200" && !strings.EqualFold(code, "success")
				}
			}
		}
	}
	return false
}

func truncate(body []byte, max int) string {
	body = bytes.TrimSpace(body)
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "..."
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

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
