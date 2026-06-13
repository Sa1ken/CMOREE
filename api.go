package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type APIClient struct {
	BaseURL string
	http    *http.Client
}

type APIError struct {
	Status  int
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewAPIClient(base string) *APIClient {
	jar, _ := cookiejar.New(nil)
	return &APIClient{
		BaseURL: normalizeBaseURL(base),
		http: &http.Client{
			Jar:     jar,
			Timeout: 25 * time.Second,
		},
	}
}

func (c *APIClient) SetBaseURL(base string) {
	c.BaseURL = normalizeBaseURL(base)
}

func normalizeBaseURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultPanelURL
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return defaultPanelURL
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func normalizeProxyURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5":
		return strings.TrimRight(parsed.String(), "/")
	default:
		return ""
	}
}

func normalizeUpdateRepo(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.TrimSuffix(value, "/")
	if strings.Contains(value, "github.com/") {
		parsed, err := url.Parse(value)
		if err == nil {
			value = strings.Trim(parsed.Path, "/")
		}
	}
	parts := strings.Split(value, "/")
	if len(parts) < 2 {
		return ""
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git")
	if owner == "" || repo == "" {
		return ""
	}
	return owner + "/" + repo
}

func (c *APIClient) URL(path string, q map[string]string) string {
	u := c.BaseURL + "/" + strings.TrimLeft(path, "/")
	if len(q) == 0 {
		return u
	}
	values := url.Values{}
	for key, value := range q {
		if value != "" {
			values.Set(key, value)
		}
	}
	if encoded := values.Encode(); encoded != "" {
		u += "?" + encoded
	}
	return u
}

func (c *APIClient) Bootstrap(ctx context.Context) (BootstrapResponse, error) {
	var out BootstrapResponse
	err := c.Get(ctx, "/api/client/bootstrap", nil, &out)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized {
			return BootstrapResponse{OK: true, Configured: true, Authenticated: false, Nav: legacyNav()}, nil
		}
		if shouldUseLegacyBootstrap(err) {
			return c.legacyBootstrap(ctx)
		}
		return out, err
	}
	return out, nil
}

func (c *APIClient) legacyBootstrap(ctx context.Context) (BootstrapResponse, error) {
	var status StatusResponse
	err := c.Get(ctx, "/api/status", nil, &status)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if apiErr.Status == http.StatusUnauthorized {
				return BootstrapResponse{OK: true, Configured: true, Authenticated: false, Nav: legacyNav()}, nil
			}
			if looksHTML([]byte(apiErr.Body)) {
				return BootstrapResponse{OK: true, Configured: false, Authenticated: false, Nav: legacyNav()}, nil
			}
		}
		return BootstrapResponse{}, err
	}
	name := status.Server.ServerName
	if name == "" {
		name = status.Server.PublicName
	}
	if name == "" {
		name = "pzserver"
	}
	return BootstrapResponse{
		OK:            true,
		Configured:    true,
		Authenticated: true,
		User: UserInfo{
			Username:    "moree",
			DisplayName: "Moree",
			RankLabel:   "Legacy admin",
			IsRoot:      true,
			Enabled:     true,
		},
		IsRoot:      true,
		Permissions: legacyPermissions(),
		Config: map[string]any{
			"server_name": name,
			"server_path": "",
		},
		Status: status,
		Nav:    legacyNav(),
	}, nil
}

func (c *APIClient) Login(ctx context.Context, username, password string) error {
	form := url.Values{}
	form.Set("username", strings.TrimSpace(username))
	form.Set("password", password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL("/login", nil), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("login: HTTP %d", resp.StatusCode)
	}
	boot, err := c.Bootstrap(ctx)
	if err != nil {
		return err
	}
	if !boot.Authenticated {
		return fmt.Errorf("неверный логин или пароль")
	}
	return nil
}

func (c *APIClient) Get(ctx context.Context, path string, q map[string]string, out any) error {
	return c.do(ctx, http.MethodGet, path, q, nil, out)
}

func (c *APIClient) Delete(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil, out)
}

func (c *APIClient) PostJSON(ctx context.Context, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	return c.do(ctx, http.MethodPost, path, nil, body, out, "application/json")
}

func (c *APIClient) do(ctx context.Context, method, path string, q map[string]string, body io.Reader, out any, contentType ...string) error {
	req, err := http.NewRequestWithContext(ctx, method, c.URL(path, q), body)
	if err != nil {
		return err
	}
	if len(contentType) > 0 && contentType[0] != "" {
		req.Header.Set("Content-Type", contentType[0])
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return &APIError{Status: resp.StatusCode, Message: "требуется вход в систему", Body: string(raw)}
	}
	if resp.StatusCode == http.StatusForbidden {
		return &APIError{Status: resp.StatusCode, Message: "недостаточно прав", Body: string(raw)}
	}
	if resp.StatusCode >= 400 {
		msg := extractAPIMessage(raw)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &APIError{Status: resp.StatusCode, Message: msg, Body: string(raw)}
	}
	if out == nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		if looksHTML(raw) {
			return &APIError{Status: resp.StatusCode, Message: htmlAPIMessage(), Body: string(raw)}
		}
		return fmt.Errorf("JSON: %w", err)
	}
	if m, ok := out.(*map[string]any); ok && m != nil {
		if okVal, exists := (*m)["ok"]; exists {
			if b, ok := okVal.(bool); ok && !b {
				msg := stringValue((*m)["msg"])
				if msg == "" {
					msg = "операция вернула ok=false"
				}
				return fmt.Errorf("%s", msg)
			}
		}
	}
	return nil
}

func (c *APIClient) UploadFile(ctx context.Context, path, filePath string, out any) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("path", path)
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		return err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/api/file-manager/upload", nil, &buf, out, writer.FormDataContentType())
}

func (c *APIClient) UploadBackup(ctx context.Context, filePath string, out any) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("archive", filepath.Base(filePath))
	if err != nil {
		return err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/api/backups/upload", nil, &buf, out, writer.FormDataContentType())
}

func extractAPIMessage(raw []byte) string {
	var m map[string]any
	if json.Unmarshal(raw, &m) == nil {
		return stringValue(m["msg"])
	}
	if looksHTML(raw) {
		return htmlAPIMessage()
	}
	return strings.TrimSpace(string(raw))
}

func shouldUseLegacyBootstrap(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusNotFound || looksHTML([]byte(apiErr.Body))
}

func looksHTML(raw []byte) bool {
	text := strings.TrimSpace(strings.ToLower(string(raw)))
	return strings.HasPrefix(text, "<!doctype html") || strings.HasPrefix(text, "<html") || strings.Contains(text, "<html")
}

func htmlAPIMessage() string {
	return "Сервер вернул HTML вместо API JSON. Если это CMoree-панель, клиент включит legacy-режим там, где возможно; для полного desktop API обновите server/app.py на сервере."
}

func legacyNav() map[string]bool {
	nav := map[string]bool{}
	for _, page := range pageDefs {
		nav[page.ID] = true
	}
	return nav
}

func legacyPermissions() []string {
	return []string{
		"server.control", "server.quit", "console.view", "console.send",
		"kits.view", "kits.edit", "kits.issue",
		"configs.view", "configs.edit", "files.view", "files.edit",
		"mods.view", "mods.edit", "logs.view", "db.view",
		"discord.view", "discord.edit", "scheduler.view", "scheduler.edit", "scheduler.run",
		"accounts.view", "accounts.identity", "accounts.permissions", "accounts.audit.view",
	}
}
