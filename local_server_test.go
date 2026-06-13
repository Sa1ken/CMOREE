package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalDashboardRenderRemovesJinjaPlaceholders(t *testing.T) {
	s := &localPanelServer{}
	html := s.renderDashboard(BootstrapResponse{
		OK:            true,
		Configured:    true,
		Authenticated: true,
		IsRoot:        true,
		User: UserInfo{
			Username:    "moree",
			DisplayName: "Тех.Админ - Кирилл",
			RankLabel:   "Администратор",
		},
		Permissions: []string{"server.control", "console.view"},
		Config: map[string]any{
			"server_name": "pzserver",
			"server_path": "/home/kirillchudnov/Zomboid",
		},
	})
	if strings.Contains(html, "{{") || strings.Contains(html, "{%") {
		t.Fatal("local dashboard still contains Jinja placeholders")
	}
	if !strings.Contains(html, "Тех.Админ - Кирилл") || !strings.Contains(html, "pzserver") {
		t.Fatal("local dashboard did not inject bootstrap values")
	}
}

func TestLocalSetupRenderRemovesJinjaPlaceholders(t *testing.T) {
	s := &localPanelServer{}
	html := s.renderSetup("", true, "secret-root-password")
	if strings.Contains(html, "{{") || strings.Contains(html, "{%") {
		t.Fatal("local setup still contains Jinja placeholders")
	}
	if !strings.Contains(html, "secret-root-password") {
		t.Fatal("local setup did not inject root password")
	}
}

func TestConnectionHTMLRequiresExternalAPIKey(t *testing.T) {
	html := connectionHTML("http://127.0.0.1:7235", "secret-api-key", "socks5://127.0.0.1:1080", "owner/repo", "")
	if !strings.Contains(html, `id="api"`) {
		t.Fatal("connection screen does not render external API input")
	}
	if strings.Contains(html, `id="proxy"`) {
		t.Fatal("connection screen should not ask the user for proxy manually")
	}
	if strings.Contains(html, `id="repo"`) {
		t.Fatal("connection screen should not show the GitHub update repo input")
	}
	if !strings.Contains(html, "cmoreeOpen(urlInput.value, apiInput.value, updateRepo)") {
		t.Fatal("connection screen does not pass external API key to shell binding")
	}
	if !strings.Contains(html, "secret-api-key") {
		t.Fatal("connection screen did not preserve configured API key")
	}
	if strings.Contains(html, "получен от API сервера: socks5://127.0.0.1:1080") || !strings.Contains(html, "owner/repo") {
		t.Fatal("connection screen did not keep update repo internal while hiding route details")
	}
}

func TestConnectionHTMLDefaultsUpdateRepoAndModal(t *testing.T) {
	html := connectionHTML(defaultPanelURL, "", "", "", "")
	if !strings.Contains(html, defaultUpdateRepo) {
		t.Fatal("connection screen does not default to the CMoree GitHub repo")
	}
	if !strings.Contains(html, `id="updateModal"`) || !strings.Contains(html, "cmoreeInstallUpdate(updateRepo)") {
		t.Fatal("connection screen does not render update install modal")
	}
	if !strings.Contains(html, "setTimeout(autoCheckUpdate") {
		t.Fatal("connection screen does not auto-check updates on start")
	}
	if strings.Contains(html, `<div class="msg">`) {
		t.Fatal("connection screen should not show the initial warning banner")
	}
	if !strings.Contains(html, `data:image/png;base64,`) {
		t.Fatal("connection screen does not embed the real logo")
	}
}

func TestAppChromeScriptIncludesWindowControls(t *testing.T) {
	script := appChromeInitScript()
	for _, piece := range []string{
		"cmoree-app-chrome",
		"cmoreeWindowMinimize",
		"cmoreeWindowToggleMaximize",
		"cmoreeWindowClose",
		"data:image/png;base64,",
	} {
		if !strings.Contains(script, piece) {
			t.Fatalf("app chrome script does not contain %q", piece)
		}
	}
}

func TestReleaseVersionComparisonIgnoresNonVersionTags(t *testing.T) {
	cases := []struct {
		latest  string
		current string
		want    bool
	}{
		{latest: "cmoree", current: "2.1.4", want: false},
		{latest: "v2.1.4", current: "2.1.4", want: false},
		{latest: "v2.1.5", current: "2.1.4", want: true},
		{latest: "cmoree-v2.1.5", current: "2.1.4", want: true},
		{latest: "v2.1.3", current: "2.1.4", want: false},
	}
	for _, tc := range cases {
		if got := isReleaseNewer(tc.latest, tc.current); got != tc.want {
			t.Fatalf("isReleaseNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestTrayNotificationCode(t *testing.T) {
	for _, code := range []uintptr{wmLButtonDown, wmLButtonUp, wmLButtonDbl, wmRButtonUp, wmContextMenu, ninSelect, ninKeySelect} {
		if got := trayNotificationCode(code); got != code {
			t.Fatalf("trayNotificationCode(%#x) = %#x", code, got)
		}
		packed := code | 0x12340000
		if got := trayNotificationCode(packed); got != code {
			t.Fatalf("trayNotificationCode(%#x) = %#x, want %#x", packed, got, code)
		}
	}
}

func TestApplyClientNetworkFromAPI(t *testing.T) {
	s := &localPanelServer{client: &http.Client{}}
	s.SetCredentials(defaultPanelURL, "secret", "")
	s.applyClientNetwork(externalPanelCheckResponse{
		OK: true,
		ClientNetwork: clientNetworkProfile{
			Mode:     "proxy",
			ProxyURL: "socks5://127.0.0.1:1080",
			Label:    "API route",
		},
	})
	if got := s.ProxyURL(); got != "socks5://127.0.0.1:1080" {
		t.Fatalf("proxy from API was not applied, got %q", got)
	}
	s.applyClientNetwork(externalPanelCheckResponse{
		OK: true,
		ClientNetwork: clientNetworkProfile{
			Mode: "direct",
		},
	})
	if got := s.ProxyURL(); got != "" {
		t.Fatalf("direct API route did not clear proxy, got %q", got)
	}
}

func TestCheckExternalAccessStopsOnLoginRedirect(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/external-panel/check" {
			http.Redirect(w, r, "/login?next=/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer backend.Close()

	s := &localPanelServer{client: &http.Client{}}
	s.SetCredentials(backend.URL, "secret", "")
	err := s.CheckExternalAccess(context.Background())
	if err == nil {
		t.Fatal("expected redirect error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "/login?next=/") || strings.Contains(msg, "stopped after 10 redirects") {
		t.Fatalf("unexpected redirect error: %s", msg)
	}
}

func TestCheckExternalAccessPrefersDirectOverSavedRoute(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external-panel/check" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-CMoree-External-API") != "secret" {
			t.Fatalf("missing external API header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"configured":true,"api_configured":true,"client_network":{"mode":"direct"}}`))
	}))
	defer backend.Close()

	s := &localPanelServer{client: &http.Client{}}
	s.SetCredentials(backend.URL, "secret", "http://127.0.0.1:1")
	if err := s.CheckExternalAccess(context.Background()); err != nil {
		t.Fatalf("direct check should ignore stale saved route: %v", err)
	}
	if got := s.ProxyURL(); got != "" {
		t.Fatalf("successful direct check should clear stale route, got %q", got)
	}
}

func TestUpdateInstallerScriptUsesPendingBackupAndRestart(t *testing.T) {
	script := updateInstallerScript(1234, `C:\tmp\new.exe`, `C:\app\CMoreeRemotePanel.exe`, `C:\tmp\work`, `C:\tmp\cmoree.log`)
	for _, piece := range []string{
		"Stop-Process -Id $TargetPid -Force",
		"CMoreeRemotePanel.exe.pending",
		"CMoreeRemotePanel.exe.bak-",
		"Move-Item -LiteralPath $Pending -Destination $Target -Force",
		"Start-Process -FilePath $Target",
		"Write-UpdateLog",
	} {
		if !strings.Contains(script, piece) {
			t.Fatalf("update helper script does not contain %q", piece)
		}
	}
}
