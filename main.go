package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	defaultPanelURL = "http://127.0.0.1:7235"
	appTitle        = "CMoree Remote Panel"
)

//go:embed assets/logo.png
var appLogoPNG []byte

var appVersion = "2.1.11"

type shellConfig struct {
	PanelURL       string `json:"panel_url"`
	ExternalAPIKey string `json:"external_api_key"`
	ProxyURL       string `json:"proxy_url"`
	UpdateRepo     string `json:"update_repo"`
}

func main() {
	cfg := initialShellConfig()
	panelURL := normalizeBaseURL(cfg.PanelURL)
	apiKey := strings.TrimSpace(cfg.ExternalAPIKey)
	proxyURL := normalizeProxyURL(cfg.ProxyURL)
	localPanel, err := startLocalPanelServer(panelURL, apiKey, proxyURL)
	if err != nil {
		messageBox("Не удалось запустить локальный HTML-сервер панели: " + err.Error())
		return
	}
	dataPath := webViewDataPath()
	clearWebViewCache(dataPath)
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  dataPath,
		WindowOptions: webview2.WindowOptions{
			Title:  appTitle,
			Width:  1480,
			Height: 900,
			IconId: 2,
			Center: true,
		},
	})
	if w == nil {
		messageBox("Не удалось открыть WebView2. Установите Microsoft Edge WebView2 Runtime и запустите EXE снова.")
		return
	}
	defer w.Destroy()
	applyDarkWindowChrome(w)

	windowShell := installAppWindowShell(w)
	if windowShell != nil {
		defer windowShell.Close()
	}
	w.Init(appChromeInitScript())

	shell := &webShell{view: w, panelURL: panelURL, apiKey: apiKey, proxyURL: proxyURL, updateRepo: normalizeUpdateRepo(cfg.UpdateRepo), local: localPanel, window: windowShell}
	shell.bind()
	shell.showConnectionPage("")
	w.SetTitle(appTitle)
	w.Run()
}

type webShell struct {
	view       webview2.WebView
	panelURL   string
	apiKey     string
	proxyURL   string
	updateRepo string
	local      *localPanelServer
	window     *appWindowShell
}

func (s *webShell) bind() {
	_ = s.view.Bind("cmoreeOpen", func(rawURL, rawAPIKey, rawUpdateRepo string) (map[string]string, error) {
		nextURL := normalizeBaseURL(rawURL)
		nextKey := strings.TrimSpace(rawAPIKey)
		nextUpdateRepo := normalizeUpdateRepo(rawUpdateRepo)
		s.panelURL = nextURL
		s.apiKey = nextKey
		s.updateRepo = nextUpdateRepo
		s.local.SetCredentials(nextURL, nextKey, s.proxyURL)
		_ = saveShellConfig(shellConfig{PanelURL: nextURL, ExternalAPIKey: nextKey, ProxyURL: s.proxyURL, UpdateRepo: nextUpdateRepo})
		if err := s.local.CheckExternalAccess(context.Background()); err != nil {
			s.view.Dispatch(func() {
				s.view.Navigate(s.local.URL())
			})
			return map[string]string{"ok": "false", "msg": err.Error(), "url": nextURL}, nil
		}
		s.proxyURL = s.local.ProxyURL()
		_ = saveShellConfig(shellConfig{PanelURL: nextURL, ExternalAPIKey: nextKey, ProxyURL: s.proxyURL, UpdateRepo: nextUpdateRepo})
		s.view.Dispatch(func() {
			s.view.Navigate(s.local.URL())
		})
		return map[string]string{"ok": "true", "url": nextURL}, nil
	})
	_ = s.view.Bind("cmoreeRetry", func() (map[string]string, error) {
		nextURL := normalizeBaseURL(s.panelURL)
		s.local.SetCredentials(nextURL, s.apiKey, s.proxyURL)
		if err := s.local.CheckExternalAccess(context.Background()); err != nil {
			s.view.Dispatch(func() {
				s.view.Navigate(s.local.URL())
			})
			return map[string]string{"ok": "false", "msg": err.Error(), "url": nextURL}, nil
		}
		s.proxyURL = s.local.ProxyURL()
		_ = saveShellConfig(shellConfig{PanelURL: s.panelURL, ExternalAPIKey: s.apiKey, ProxyURL: s.proxyURL, UpdateRepo: s.updateRepo})
		s.view.Dispatch(func() {
			s.view.Navigate(s.local.URL())
		})
		return map[string]string{"ok": "true", "url": nextURL}, nil
	})
	_ = s.view.Bind("cmoreeForceOpen", func(rawURL, rawAPIKey, rawUpdateRepo string) (map[string]string, error) {
		nextURL := normalizeBaseURL(rawURL)
		nextKey := strings.TrimSpace(rawAPIKey)
		nextUpdateRepo := normalizeUpdateRepo(rawUpdateRepo)
		s.panelURL = nextURL
		s.apiKey = nextKey
		s.updateRepo = nextUpdateRepo
		s.local.SetCredentials(nextURL, nextKey, s.proxyURL)
		_ = saveShellConfig(shellConfig{PanelURL: nextURL, ExternalAPIKey: nextKey, ProxyURL: s.proxyURL, UpdateRepo: nextUpdateRepo})
		if err := s.local.CheckExternalAccess(context.Background()); err != nil {
			s.view.Dispatch(func() {
				s.view.Navigate(s.local.URL())
			})
			return map[string]string{"ok": "false", "msg": err.Error(), "url": nextURL}, nil
		}
		s.proxyURL = s.local.ProxyURL()
		_ = saveShellConfig(shellConfig{PanelURL: nextURL, ExternalAPIKey: nextKey, ProxyURL: s.proxyURL, UpdateRepo: nextUpdateRepo})
		s.view.Dispatch(func() {
			s.view.Navigate(s.local.URL())
		})
		return map[string]string{"ok": "true", "url": nextURL}, nil
	})
	_ = s.view.Bind("cmoreeReset", func() (map[string]string, error) {
		s.panelURL = defaultPanelURL
		s.apiKey = ""
		s.proxyURL = ""
		s.updateRepo = defaultUpdateRepo
		s.local.SetCredentials(defaultPanelURL, "", "")
		_ = saveShellConfig(shellConfig{PanelURL: defaultPanelURL, ExternalAPIKey: "", ProxyURL: "", UpdateRepo: defaultUpdateRepo})
		s.view.Dispatch(func() {
			s.view.Navigate(s.local.URL())
		})
		return map[string]string{"ok": "true", "url": defaultPanelURL}, nil
	})
	_ = s.view.Bind("cmoreeCheckUpdate", func(rawUpdateRepo string) (map[string]string, error) {
		return s.checkUpdateResult(rawUpdateRepo), nil
	})
	_ = s.view.Bind("cmoreeInstallUpdate", func(rawUpdateRepo string) (map[string]string, error) {
		return s.installUpdate(rawUpdateRepo)
	})
	_ = s.view.Bind("cmoreeWindowMinimize", func() error {
		if s.window != nil {
			s.window.Minimize()
		}
		return nil
	})
	_ = s.view.Bind("cmoreeWindowToggleMaximize", func() error {
		if s.window != nil {
			s.window.ToggleMaximize()
		}
		return nil
	})
	_ = s.view.Bind("cmoreeWindowClose", func() error {
		if s.window != nil {
			s.window.HideToTray()
		}
		return nil
	})
}

func (s *webShell) showConnectionPage(message string) {
	s.view.SetTitle(appTitle)
	s.view.SetHtml(connectionHTML(s.panelURL, s.apiKey, s.proxyURL, s.updateRepo, message))
}

func initialShellConfig() shellConfig {
	cfg := shellConfig{PanelURL: defaultPanelURL}
	if saved, err := loadShellConfig(); err == nil {
		cfg = saved
	}
	if envURL := strings.TrimSpace(os.Getenv("CMOREE_PANEL_URL")); envURL != "" {
		cfg.PanelURL = envURL
	}
	if envKey := strings.TrimSpace(os.Getenv("CMOREE_EXTERNAL_API_KEY")); envKey != "" {
		cfg.ExternalAPIKey = envKey
	}
	if envProxy := strings.TrimSpace(os.Getenv("CMOREE_PROXY_URL")); envProxy != "" {
		cfg.ProxyURL = envProxy
	}
	if envRepo := strings.TrimSpace(os.Getenv("CMOREE_UPDATE_REPO")); envRepo != "" {
		cfg.UpdateRepo = envRepo
	}
	if len(os.Args) > 1 {
		cfg.PanelURL = os.Args[1]
	}
	cfg.PanelURL = normalizeBaseURL(cfg.PanelURL)
	cfg.ExternalAPIKey = strings.TrimSpace(cfg.ExternalAPIKey)
	cfg.ProxyURL = normalizeProxyURL(cfg.ProxyURL)
	cfg.UpdateRepo = normalizeUpdateRepo(cfg.UpdateRepo)
	if cfg.UpdateRepo == "" {
		cfg.UpdateRepo = defaultUpdateRepo
	}
	return cfg
}

func initialPanelURL() string {
	return initialShellConfig().PanelURL
}

func pingPanel(rawURL string) error {
	rawURL = normalizeBaseURL(rawURL)
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CMoreeRemotePanel/2 WebView2Shell")
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("сервер %s ответил HTTP %d", rawURL, resp.StatusCode)
	}
	return nil
}

func loadShellConfig() (shellConfig, error) {
	path, err := shellConfigPath()
	if err != nil {
		return shellConfig{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return shellConfig{}, err
	}
	var cfg shellConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return shellConfig{}, err
	}
	return cfg, nil
}

func saveShellConfig(cfg shellConfig) error {
	path, err := shellConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func shellConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "CMoreeRemotePanel", "config.json"), nil
}

func webViewDataPath() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	path := filepath.Join(base, "CMoreeRemotePanel", "WebView2")
	_ = os.MkdirAll(path, 0o755)
	return path
}

func clearWebViewCache(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	_ = os.RemoveAll(path)
	_ = os.MkdirAll(path, 0o755)
}

func appLogoDataURL() string {
	if len(appLogoPNG) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(appLogoPNG)
}

func connectionHTML(panelURL, apiKey, proxyURL, updateRepo, message string) string {
	panelURL = template.HTMLEscapeString(normalizeBaseURL(panelURL))
	apiKey = template.HTMLEscapeString(strings.TrimSpace(apiKey))
	normalizedRepo := normalizeUpdateRepo(updateRepo)
	if normalizedRepo == "" {
		normalizedRepo = defaultUpdateRepo
	}
	updateRepo = template.HTMLEscapeString(normalizedRepo)
	message = template.HTMLEscapeString(strings.TrimSpace(message))
	msgHTML := ""
	if message != "" {
		msgHTML = `<div class="msg">` + message + `</div>`
	}
	logoDataURL := template.HTMLEscapeString(appLogoDataURL())
	return `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CMoree Remote Panel</title>
<style>
:root{color-scheme:dark;--bg:#0d141d;--top:#0f1823;--panel:#132130;--panel2:#18293b;--text:#eff5fb;--muted:#8fa3bc;--dim:#677a91;--accent:#73aeff;--good:#68c49a;--bad:#e5848c;--warn:#efb86c;--bd:rgba(157,180,209,.22)}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:var(--bg);font:14px/1.45 "Manrope","Segoe UI",Arial,sans-serif;color:var(--text);display:grid;place-items:center;padding:28px}
.card{width:min(680px,100%);background:var(--panel);border:1px solid var(--bd);border-radius:18px;padding:28px;box-shadow:0 24px 70px rgba(0,0,0,.35)}
.brand{display:flex;align-items:center;gap:12px;margin-bottom:24px}.mark{width:46px;height:46px;display:grid;place-items:center}.mark img{display:block;width:46px;height:46px;object-fit:contain}.brand h1{margin:0;font-size:24px}.brand p{margin:2px 0 0;color:var(--muted);font-size:13px}
.msg{border:1px solid rgba(229,132,140,.26);background:rgba(229,132,140,.16);color:#ffb5bb;border-radius:14px;padding:13px 14px;margin-bottom:16px;white-space:pre-wrap}
label{display:block;color:var(--muted);font-size:12px;margin:0 0 7px 2px}input{width:100%;border:0;outline:0;border-radius:13px;padding:13px 14px;background:rgba(255,255,255,.1);color:var(--text);font:inherit;margin-bottom:14px}input::placeholder{color:var(--dim)}
.row{display:flex;gap:9px;flex-wrap:wrap}button{border:0;border-radius:13px;padding:11px 15px;color:var(--text);background:rgba(255,255,255,.1);font-weight:700;cursor:pointer}button:hover{filter:brightness(1.08)}button:disabled{opacity:.58;cursor:default;filter:none}button.primary{background:rgba(115,174,255,.32)}button.good{background:rgba(104,196,154,.24);color:var(--good)}button.warn{background:rgba(239,184,108,.22);color:var(--warn)}button.bad{background:rgba(229,132,140,.22);color:#ffb5bb}
.hint{margin-top:18px;color:var(--muted);font-size:12px}.hint code{color:var(--accent);font-family:"JetBrains Mono",Consolas,monospace}.status{margin-top:12px;color:var(--dim);min-height:18px;font-size:12px}.status.warn{color:var(--warn)}
.modal{position:fixed;inset:0;display:none;align-items:center;justify-content:center;background:rgba(5,10,16,.72);padding:24px;z-index:20}.modal.on{display:flex}.dialog{width:min(620px,100%);background:#111d2a;border:1px solid var(--bd);border-radius:20px;padding:22px;box-shadow:0 28px 90px rgba(0,0,0,.55)}.dialog h2{margin:0 0 8px;font-size:24px}.dialog .meta{color:var(--muted);margin-bottom:14px}.release{max-height:260px;overflow:auto;white-space:pre-wrap;background:rgba(255,255,255,.05);border:1px solid rgba(157,180,209,.14);border-radius:14px;padding:14px;color:#dbe6f3;font-family:"JetBrains Mono",Consolas,monospace;font-size:12px}.dialog-actions{display:flex;justify-content:flex-end;gap:10px;margin-top:18px}.pill{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--bd);border-radius:999px;padding:5px 10px;color:var(--muted);font-size:12px;margin-bottom:12px}
</style>
</head>
<body>
<main class="card">
 <div class="brand"><div class="mark"><img src="` + logoDataURL + `" alt=""></div><div><h1>Moree</h1><p>Панель удалённого управления · v` + template.HTMLEscapeString(appVersion) + `</p></div></div>
 ` + msgHTML + `
 <label for="url">Адрес CMoree панели</label>
 <input id="url" value="` + panelURL + `" spellcheck="false" autocomplete="off">
 <label for="api">API-ключ аккаунта</label>
 <input id="api" value="` + apiKey + `" type="password" spellcheck="false" autocomplete="off">
 <div class="row">
  <button class="primary" onclick="openPanel()">Открыть</button>
  <button onclick="retryPanel()">Проверить снова</button>
  <button class="good" onclick="forceOpen()">Открыть без проверки</button>
  <button onclick="checkUpdate()">Проверить обновления</button>
  <button class="warn" onclick="resetUrl()">127.0.0.1</button>
 </div>
 <div id="status" class="status"></div>
</main>
<section id="updateModal" class="modal" aria-hidden="true">
 <div class="dialog">
  <span class="pill">GitHub Releases</span>
  <h2>Доступно обновление</h2>
  <div id="updateMeta" class="meta"></div>
  <div id="updateBody" class="release"></div>
  <div class="dialog-actions">
   <button class="bad" onclick="cancelUpdate()">Отмена</button>
   <button id="installBtn" class="good" onclick="installUpdate()">Установить</button>
  </div>
 </div>
</section>
<script>
const urlInput = document.getElementById('url');
const apiInput = document.getElementById('api');
const updateRepo = '` + updateRepo + `';
const statusEl = document.getElementById('status');
const updateModal = document.getElementById('updateModal');
const updateMeta = document.getElementById('updateMeta');
const updateBody = document.getElementById('updateBody');
const installBtn = document.getElementById('installBtn');
let latestUpdate = null;
function setStatus(text, tone){
 statusEl.textContent = text || '';
 statusEl.className = 'status' + (tone ? ' ' + tone : '');
}
async function openPanel(){
 setStatus('Проверяю подключение...');
 const r = await window.cmoreeOpen(urlInput.value, apiInput.value, updateRepo);
 if(r && r.ok === 'false') setStatus(r.msg || 'Не удалось подключиться.');
}
async function retryPanel(){
 setStatus('Проверяю сохранённый адрес...');
 const r = await window.cmoreeRetry();
 if(r && r.ok === 'false') setStatus(r.msg || 'Не удалось подключиться.');
}
async function forceOpen(){
 setStatus('Открываю без проверки...');
 const r = await window.cmoreeForceOpen(urlInput.value, apiInput.value, updateRepo);
 if(r && r.ok === 'false') setStatus(r.msg || 'Не удалось подключиться.');
}
async function checkUpdate(){
 setStatus('Проверяю GitHub Releases...');
 const r = await window.cmoreeCheckUpdate(updateRepo);
 if(!r) return;
 if(r.ok === 'false') setStatus(r.msg || 'Не удалось проверить обновления.');
 else if(r.update === 'true') showUpdateModal(r, false);
 else setStatus(r.msg || 'Установлена актуальная версия.');
}
async function autoCheckUpdate(){
 const r = await window.cmoreeCheckUpdate(updateRepo);
 if(r && r.ok === 'true' && r.update === 'true') showUpdateModal(r, true);
}
function showUpdateModal(r, automatic){
 latestUpdate = r;
 const asset = r.asset ? ' · ассет: ' + r.asset : '';
 updateMeta.textContent = 'Текущая: ' + (r.current || '?') + ' · новая: ' + (r.version || '?') + asset;
 updateBody.textContent = r.description || 'Описание релиза не заполнено.';
 installBtn.disabled = false;
 installBtn.textContent = 'Установить';
 updateModal.classList.add('on');
 updateModal.setAttribute('aria-hidden', 'false');
 setStatus((automatic ? 'При запуске найдено обновление ' : 'Доступно обновление ') + (r.version || '') + '.', 'warn');
}
function cancelUpdate(){
 updateModal.classList.remove('on');
 updateModal.setAttribute('aria-hidden', 'true');
 setStatus('Можно продолжить работу, но некоторые функции могут быть недоступны без последнего обновления.', 'warn');
}
async function installUpdate(){
 installBtn.disabled = true;
 installBtn.textContent = 'Скачиваю...';
 setStatus('Скачиваю и устанавливаю обновление...');
 if(updateBody) updateBody.textContent = (latestUpdate && latestUpdate.description ? latestUpdate.description + '\n\n' : '') + 'Скачивание началось. После загрузки приложение закроется и заменит EXE.';
 let r = null;
 try {
  r = await window.cmoreeInstallUpdate(updateRepo);
 } catch(e) {
  installBtn.disabled = false;
  installBtn.textContent = 'Установить';
  setStatus('Ошибка запуска обновления: ' + (e && e.message ? e.message : String(e || 'unknown')), 'warn');
  return;
 }
 if(!r || r.ok === 'false'){
  installBtn.disabled = false;
  installBtn.textContent = 'Установить';
  setStatus((r && r.msg) || 'Не удалось установить обновление.', 'warn');
  return;
 }
 installBtn.textContent = 'Закрываю...';
 setStatus(r.msg || 'Обновление скачано. Приложение перезапустится автоматически.');
}
async function resetUrl(){
 const r = await window.cmoreeReset();
 if(r && r.url) urlInput.value = r.url;
 apiInput.value = '';
 setStatus('Адрес сброшен.');
}
urlInput.addEventListener('keydown', e => { if(e.key === 'Enter') openPanel(); });
apiInput.addEventListener('keydown', e => { if(e.key === 'Enter') openPanel(); });
setTimeout(autoCheckUpdate, 900);
</script>
</body>
</html>`
}

func cleanMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "Панель пока недоступна."
	}
	return message
}

func messageBox(text string) {
	titlePtr, _ := windows.UTF16PtrFromString(appTitle)
	textPtr, _ := windows.UTF16PtrFromString(text)
	_, _ = windows.MessageBox(0, textPtr, titlePtr, windows.MB_ICONERROR|windows.MB_OK)
}
