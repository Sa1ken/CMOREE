package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed local_web/templates/* local_web/static/*
var localWeb embed.FS

type localPanelServer struct {
	mu        sync.RWMutex
	backend   *url.URL
	apiKey    string
	proxyURL  string
	jar       http.CookieJar
	client    *http.Client
	transport *http.Transport
	server    *http.Server
	localURL  string
}

type externalPanelCheckResponse struct {
	OK             bool                 `json:"ok"`
	Msg            string               `json:"msg"`
	Configured     bool                 `json:"configured"`
	APIConfigured  bool                 `json:"api_configured"`
	ClientNetwork  clientNetworkProfile `json:"client_network"`
	LegacyProxyURL string               `json:"proxy_url"`
}

type clientNetworkProfile struct {
	Mode       string `json:"mode"`
	Configured bool   `json:"configured"`
	ProxyURL   string `json:"proxy_url"`
	VPNURL     string `json:"vpn_url"`
	Label      string `json:"label"`
	Source     string `json:"source"`
}

func startLocalPanelServer(backendURL, apiKey, proxyURL string) (*localPanelServer, error) {
	jar, _ := cookiejar.New(nil)
	s := &localPanelServer{
		jar:    jar,
		client: &http.Client{Timeout: 25 * time.Second, Jar: jar},
	}
	s.SetCredentials(backendURL, apiKey, proxyURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.server = &http.Server{Handler: mux}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s.localURL = "http://" + listener.Addr().String()
	go func() {
		_ = s.server.Serve(listener)
	}()
	return s, nil
}

func (s *localPanelServer) URL() string {
	return s.localURL
}

func (s *localPanelServer) SetBackend(rawURL string) {
	s.SetCredentials(rawURL, s.APIKey(), s.ProxyURL())
}

func (s *localPanelServer) SetCredentials(rawURL, apiKey, proxyURL string) {
	parsed, err := url.Parse(normalizeBaseURL(rawURL))
	if err != nil {
		parsed, _ = url.Parse(defaultPanelURL)
	}
	jar, _ := cookiejar.New(nil)
	transport := newProxyTransport(proxyURL)
	s.mu.Lock()
	s.backend = parsed
	s.apiKey = strings.TrimSpace(apiKey)
	s.proxyURL = normalizeProxyURL(proxyURL)
	s.jar = jar
	s.client.Jar = jar
	s.transport = transport
	s.client.Transport = transport
	s.mu.Unlock()
}

func (s *localPanelServer) SetProxy(proxyURL string) {
	transport := newProxyTransport(proxyURL)
	s.mu.Lock()
	s.proxyURL = normalizeProxyURL(proxyURL)
	s.transport = transport
	s.client.Transport = transport
	s.mu.Unlock()
}

func (s *localPanelServer) Backend() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.backend == nil {
		return defaultPanelURL
	}
	return s.backend.String()
}

func (s *localPanelServer) APIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.apiKey)
}

func (s *localPanelServer) ProxyURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.proxyURL)
}

func (s *localPanelServer) hasExternalCredentials() bool {
	return s.APIKey() != ""
}

func (s *localPanelServer) backendURL(path string, q string) string {
	s.mu.RLock()
	var base url.URL
	if s.backend != nil {
		base = *s.backend
	} else {
		parsed, _ := url.Parse(defaultPanelURL)
		base = *parsed
	}
	s.mu.RUnlock()
	base.Path = singleJoiningSlash(base.Path, path)
	base.RawQuery = q
	return base.String()
}

func (s *localPanelServer) addExternalHeaders(req *http.Request) {
	req.Header.Set("X-CMoree-External-Panel", "1")
	if key := s.APIKey(); key != "" {
		req.Header.Set("X-CMoree-External-API", key)
	}
}

func newProxyTransport(proxyValue string) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	proxyValue = normalizeProxyURL(proxyValue)
	if proxyValue == "" {
		return transport
	}
	parsed, err := url.Parse(proxyValue)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return transport
	}
	transport.Proxy = http.ProxyURL(parsed)
	return transport
}

func (s *localPanelServer) CheckExternalAccess(ctx context.Context) error {
	if !s.hasExternalCredentials() {
		return fmt.Errorf("сначала укажите IP/URL сервера и API-ключ внешней панели")
	}
	savedRoute := s.ProxyURL()
	if savedRoute != "" {
		s.SetProxy("")
	}
	directErr := s.checkExternalAccessOnce(ctx)
	if directErr == nil {
		return nil
	}
	if savedRoute != "" {
		s.SetProxy(savedRoute)
		routeErr := s.checkExternalAccessOnce(ctx)
		if routeErr == nil {
			return nil
		}
		s.SetProxy("")
		return fmt.Errorf("direct: %v; saved route %s: %v", directErr, savedRoute, routeErr)
	}
	return directErr
}

func (s *localPanelServer) checkExternalAccessOnce(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	payload, _ := json.Marshal(map[string]string{"api_key": s.APIKey()})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.backendURL("/api/external-panel/check", ""), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CMoreeRemotePanel/2 LocalHTML")
	s.addExternalHeaders(req)
	client := *s.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к backend %s: %w", s.Backend(), err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			location = "redirect"
		}
		return fmt.Errorf("backend %s перенаправил API-проверку на %s. Обновите server/app.py на сервере: EXE должен входить по аккаунтному API-ключу, без /login-редиректа", s.Backend(), location)
	}
	var check externalPanelCheckResponse
	_ = json.Unmarshal(raw, &check)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if resp.StatusCode >= 400 || !boolFromAny(out["ok"]) {
		msg := strings.TrimSpace(check.Msg)
		if msg == "" {
			msg = stringValue(out["msg"])
		}
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("%s", msg)
	}
	s.applyClientNetwork(check)
	return nil
}

func (s *localPanelServer) applyClientNetwork(check externalPanelCheckResponse) {
	proxy := normalizeProxyURL(check.ClientNetwork.ProxyURL)
	if proxy == "" && strings.EqualFold(check.ClientNetwork.Mode, "vpn") {
		proxy = normalizeProxyURL(check.ClientNetwork.VPNURL)
	}
	if proxy == "" {
		proxy = normalizeProxyURL(check.LegacyProxyURL)
	}
	if proxy != s.ProxyURL() {
		s.SetProxy(proxy)
	}
}

func (s *localPanelServer) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/static/"):
		s.serveStatic(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/socket.io/"):
		s.proxy(w, r)
	case r.URL.Path == "/" || r.URL.Path == "/dashboard":
		s.handleDashboard(w, r)
	case r.URL.Path == "/login":
		s.handleLogin(w, r)
	case r.URL.Path == "/logout":
		s.handleLogout(w, r)
	case r.URL.Path == "/setup":
		s.handleSetup(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *localPanelServer) serveStatic(w http.ResponseWriter, r *http.Request) {
	staticFS, err := fs.Sub(localWeb, "local_web/static")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))).ServeHTTP(w, r)
}

func (s *localPanelServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if !s.hasExternalCredentials() {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", "Сначала укажите IP/URL сервера и API-ключ внешней панели. Ключ появится в модальном окне обычной web-панели сервера, если его ещё нет в конфиге."))
		return
	}
	if err := s.CheckExternalAccess(r.Context()); err != nil {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", err.Error()))
		return
	}
	boot, err := s.bootstrap(r.Context())
	if err != nil {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", err.Error()))
		return
	}
	if !boot.Configured {
		s.writeHTML(w, s.renderSetup("", false, ""))
		return
	}
	if !boot.Authenticated {
		s.writeHTML(w, s.renderLogin(""))
		return
	}
	s.writeHTML(w, s.renderDashboard(boot))
}

func (s *localPanelServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.hasExternalCredentials() {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", "Сначала укажите IP/URL сервера и API-ключ внешней панели."))
		return
	}
	if err := s.CheckExternalAccess(r.Context()); err != nil {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", err.Error()))
		return
	}
	if r.Method == http.MethodGet {
		s.writeHTML(w, s.renderLogin(""))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.writeHTML(w, s.renderLogin("Не удалось прочитать форму входа"))
		return
	}
	form := url.Values{}
	form.Set("username", r.FormValue("username"))
	form.Set("password", r.FormValue("password"))
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.backendURL("/login", ""), strings.NewReader(form.Encode()))
	if err != nil {
		s.writeHTML(w, s.renderLogin(err.Error()))
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "CMoreeRemotePanel/2 LocalHTML")
	s.addExternalHeaders(req)

	client := *s.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		s.writeHTML(w, s.renderLogin("Backend недоступен: "+err.Error()))
		return
	}
	_ = resp.Body.Close()

	boot, err := s.bootstrap(r.Context())
	if err == nil && boot.Authenticated {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.writeHTML(w, s.renderLogin("Неверный логин или пароль"))
}

func (s *localPanelServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.backendURL("/logout", ""), nil)
	if err == nil {
		s.addExternalHeaders(req)
		_, _ = s.client.Do(req)
	}
	jar, _ := cookiejar.New(nil)
	s.jar = jar
	s.client.Jar = jar
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *localPanelServer) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !s.hasExternalCredentials() {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", "Сначала укажите IP/URL сервера и API-ключ внешней панели."))
		return
	}
	if err := s.CheckExternalAccess(r.Context()); err != nil {
		s.writeHTML(w, connectionHTML(s.Backend(), s.APIKey(), s.ProxyURL(), "", err.Error()))
		return
	}
	if r.Method == http.MethodGet {
		s.writeHTML(w, s.renderSetup("", false, ""))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.writeHTML(w, s.renderSetup("Не удалось прочитать форму настройки", false, ""))
		return
	}
	payload := map[string]string{}
	for key, values := range r.PostForm {
		if len(values) > 0 {
			payload[key] = values[0]
		}
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.backendURL("/api/client/setup", ""), bytes.NewReader(raw))
	if err != nil {
		s.writeHTML(w, s.renderSetup(err.Error(), false, ""))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CMoreeRemotePanel/2 LocalHTML")
	s.addExternalHeaders(req)
	resp, err := s.client.Do(req)
	if err != nil {
		s.writeHTML(w, s.renderSetup("Backend недоступен: "+err.Error(), false, ""))
		return
	}
	defer resp.Body.Close()
	var out map[string]any
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &out)
	if resp.StatusCode >= 400 || !boolFromAny(out["ok"]) {
		msg := stringValue(out["msg"])
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		s.writeHTML(w, s.renderSetup(msg, false, ""))
		return
	}
	s.writeHTML(w, s.renderSetup("", true, stringValue(out["root_password"])))
}

func (s *localPanelServer) bootstrap(ctx context.Context) (BootstrapResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.backendURL("/api/client/bootstrap", ""), nil)
	if err != nil {
		return BootstrapResponse{}, err
	}
	req.Header.Set("User-Agent", "CMoreeRemotePanel/2 LocalHTML")
	s.addExternalHeaders(req)
	resp, err := s.client.Do(req)
	if err != nil {
		return BootstrapResponse{}, fmt.Errorf("не удалось подключиться к backend %s: %w", s.Backend(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return BootstrapResponse{OK: true, Configured: true, Authenticated: false}, nil
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return BootstrapResponse{}, fmt.Errorf("backend ответил HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var boot BootstrapResponse
	if err := json.Unmarshal(raw, &boot); err != nil {
		return BootstrapResponse{}, fmt.Errorf("backend вернул не JSON bootstrap: %w", err)
	}
	return boot, nil
}

func (s *localPanelServer) proxy(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	target := *s.backend
	transport := s.transport
	s.mu.RUnlock()
	proxy := httputil.NewSingleHostReverseProxy(&target)
	original := proxy.Director
	proxy.Director = func(req *http.Request) {
		original(req)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		req.Header.Set("User-Agent", "CMoreeRemotePanel/2 LocalHTML")
		req.Header.Set("X-CMoree-Desktop-Local", "1")
		s.addExternalHeaders(req)
	}
	proxy.Transport = &jarTransport{base: transport, jar: s.jar}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Set-Cookie")
		if loc := resp.Header.Get("Location"); loc != "" {
			resp.Header.Set("Location", rewriteLocation(loc, &target))
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Backend недоступен: "+err.Error(), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

type jarTransport struct {
	base http.RoundTripper
	jar  http.CookieJar
}

func (t *jarTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.base == nil {
		t.base = http.DefaultTransport
	}
	for _, cookie := range t.jar.Cookies(req.URL) {
		req.AddCookie(cookie)
	}
	resp, err := t.base.RoundTrip(req)
	if resp != nil {
		t.jar.SetCookies(req.URL, resp.Cookies())
	}
	return resp, err
}

func (s *localPanelServer) writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, body)
}

func (s *localPanelServer) renderDashboard(boot BootstrapResponse) string {
	tpl := readLocalTemplate("dashboard.html")
	cfg := boot.Config
	serverName := firstNonEmptyString(stringValue(cfg["server_name"]), "pzserver")
	serverPath := stringValue(cfg["server_path"])
	user := boot.User
	displayName := firstNonEmptyString(user.DisplayName, user.Username, "root")
	username := firstNonEmptyString(user.Username, "root")
	rank := firstNonEmptyString(user.RankLabel, "ROOT")
	userJSON, _ := json.Marshal(user)
	permsJSON, _ := json.Marshal(boot.Permissions)
	roleClass := ""
	if boot.IsRoot {
		roleClass = " root"
	}
	roleBadge := fmt.Sprintf(`<span class="role-badge%s">%s</span>`, roleClass, html.EscapeString(rank))
	if username == "" {
		roleBadge = ""
	}
	replacements := map[string]string{
		"{{ config.server_name }}":                              html.EscapeString(serverName),
		"{{ config.server_path }}":                              html.EscapeString(serverPath),
		"{{ auth_user.display_name if auth_user else 'root' }}": html.EscapeString(displayName),
		"{{ auth_user.username if auth_user else 'root' }}":     html.EscapeString(username),
		"{{ url_for('logout') }}":                               "/logout",
		"{{ auth_user|tojson }}":                                string(userJSON),
		"{{ auth_permissions|tojson }}":                         string(permsJSON),
		"{{ 'true' if auth_is_root else 'false' }}":             boolJS(boot.IsRoot),
		`<script src="https://cdnjs.cloudflare.com/ajax/libs/socket.io/4.7.5/socket.io.min.js"></script>`: `<script src="https://cdnjs.cloudflare.com/ajax/libs/socket.io/4.7.5/socket.io.min.js"></script><script>window.io=window.io||function(){return{on:function(){},emit:function(){},disconnect:function(){},connect:function(){}}};</script>`,
	}
	for from, to := range replacements {
		tpl = strings.ReplaceAll(tpl, from, to)
	}
	tpl = regexp.MustCompile(`(?s)\s*\{% if external_panel_api_key_new and external_panel_api_key %\}.*?\{% endif %\}`).ReplaceAllString(tpl, "")
	tpl = regexp.MustCompile(`\s*\{% if auth_user %\}<span class="role-badge\{% if auth_is_root %\} root\{% endif %\}">\{\{ auth_user\.rank_label if auth_user\.rank_label else 'ROOT' \}\}</span>\{% endif %\}`).ReplaceAllString(tpl, roleBadge)
	return tpl
}

func (s *localPanelServer) renderLogin(errorText string) string {
	tpl := readLocalTemplate("login.html")
	serverName := "pzserver"
	if boot, err := s.bootstrap(context.Background()); err == nil {
		serverName = firstNonEmptyString(stringValue(boot.Config["server_name"]), serverName)
	}
	tpl = strings.ReplaceAll(tpl, "{{ config.server_name if config and config.server_name else 'pzserver' }}", html.EscapeString(serverName))
	tpl = replaceJinjaIfBlock(tpl, "error", errorHTML(errorText))
	return tpl
}

func (s *localPanelServer) renderSetup(errorText string, complete bool, rootPassword string) string {
	tpl := readLocalTemplate("setup.html")
	tpl = strings.ReplaceAll(tpl, "{{ url_for('login') }}", "/login")
	tpl = strings.ReplaceAll(tpl, "{{ root_password }}", html.EscapeString(rootPassword))
	tpl = replaceJinjaIfBlock(tpl, "error", errorHTML(errorText))
	secret := regexp.MustCompile(`(?s)\s*\{% if setup_complete and root_password %\}.*?\{% endif %\}`)
	if complete && strings.TrimSpace(rootPassword) != "" {
		match := secret.FindString(tpl)
		match = strings.ReplaceAll(match, "{% if setup_complete and root_password %}", "")
		match = strings.ReplaceAll(match, "{% endif %}", "")
		tpl = secret.ReplaceAllString(tpl, match)
	} else {
		tpl = secret.ReplaceAllString(tpl, "")
	}
	return tpl
}

func readLocalTemplate(name string) string {
	raw, err := localWeb.ReadFile("local_web/templates/" + name)
	if err != nil {
		return "<!doctype html><meta charset=\"utf-8\"><pre>" + html.EscapeString(err.Error()) + "</pre>"
	}
	return string(raw)
}

func replaceJinjaIfBlock(tpl, key, replacement string) string {
	re := regexp.MustCompile(`(?s)\s*\{% if ` + regexp.QuoteMeta(key) + ` %\}.*?\{% endif %\}`)
	if strings.TrimSpace(replacement) == "" {
		return re.ReplaceAllString(tpl, "")
	}
	return re.ReplaceAllString(tpl, replacement)
}

func errorHTML(errorText string) string {
	if strings.TrimSpace(errorText) == "" {
		return ""
	}
	return `<div class="err"><i class="fas fa-circle-exclamation"></i><div>` + html.EscapeString(errorText) + `</div></div>`
}

func boolJS(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func rewriteLocation(loc string, target *url.URL) string {
	parsed, err := url.Parse(loc)
	if err != nil {
		return loc
	}
	if parsed.IsAbs() && strings.EqualFold(parsed.Host, target.Host) {
		return singleJoiningSlash("/", parsed.Path)
	}
	return loc
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}
