package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBootstrapFallsBackWhenDesktopEndpointReturnsHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/client/bootstrap":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><html lang=en><body>dashboard</body></html>"))
		case "/api/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"running": true,
				"server": {
					"server_name": "Project Phoenix",
					"ip": "89.111.142.8",
					"players_online": 2,
					"players_max": 45
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	boot, err := client.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}
	if !boot.Authenticated || !boot.Configured {
		t.Fatalf("expected authenticated configured legacy bootstrap, got authenticated=%v configured=%v", boot.Authenticated, boot.Configured)
	}
	if boot.Config["server_name"] != "Project Phoenix" {
		t.Fatalf("unexpected server name: %#v", boot.Config["server_name"])
	}
	if !boot.Status.Running {
		t.Fatal("expected status from legacy /api/status")
	}
}

func TestBootstrapTreatsLegacyUnauthorizedAsLoginRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/client/bootstrap":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<!doctype html><html><body>login</body></html>"))
		case "/api/status":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ok": false, "msg": "login required"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	boot, err := client.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}
	if boot.Authenticated {
		t.Fatal("expected unauthenticated bootstrap")
	}
	if !boot.Configured {
		t.Fatal("expected legacy server to be treated as configured when /api/status asks for login")
	}
}

func TestHTMLAPIErrorDoesNotExposePageBodyAsToast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><html lang=en><body>not json</body></html>"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	var out map[string]any
	err := client.Get(context.Background(), "/api/client/bootstrap", nil, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "<!doctype") || strings.Contains(err.Error(), "<html") {
		t.Fatalf("HTML leaked into error text: %q", err.Error())
	}
}

func TestNormalizeBaseURLDropsLoginPathAndQuery(t *testing.T) {
	cases := map[string]string{
		"89.111.142.8:7235/login?next=/":                         "http://89.111.142.8:7235",
		"http://89.111.142.8:7235/login?next=/":                  "http://89.111.142.8:7235",
		"http://89.111.142.8:7235/dashboard#accounts":            "http://89.111.142.8:7235",
		"http://89.111.142.8:7235/api/external-panel/check":      "http://89.111.142.8:7235",
		"https://example.com/panel/login?next=%2Fdashboard#hash": "https://example.com",
	}
	for input, want := range cases {
		if got := normalizeBaseURL(input); got != want {
			t.Fatalf("normalizeBaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}
