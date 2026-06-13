package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
)

const defaultUpdateRepo = "Sa1ken/CMOREE"
const windowsCreateNoWindow = 0x08000000

type githubReleaseInfo struct {
	TagName string               `json:"tag_name"`
	Name    string               `json:"name"`
	Body    string               `json:"body"`
	HTMLURL string               `json:"html_url"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

var releaseVersionPattern = regexp.MustCompile(`(?i)\bv?(\d+)\.(\d+)\.(\d+)(?:[-+][0-9A-Za-z.-]+)?\b`)

func (s *webShell) checkUpdateResult(rawUpdateRepo string) map[string]string {
	repo := normalizeUpdateRepo(rawUpdateRepo)
	if repo == "" {
		repo = defaultUpdateRepo
	}
	s.updateRepo = repo
	_ = saveShellConfig(shellConfig{PanelURL: s.panelURL, ExternalAPIKey: s.apiKey, ProxyURL: s.proxyURL, UpdateRepo: repo})

	info, err := checkGitHubLatestRelease(context.Background(), repo, s.proxyURL)
	if err != nil {
		return map[string]string{"ok": "false", "msg": err.Error()}
	}
	return releaseResultMap(info)
}

func (s *webShell) installUpdate(rawUpdateRepo string) (map[string]string, error) {
	repo := normalizeUpdateRepo(rawUpdateRepo)
	if repo == "" {
		repo = defaultUpdateRepo
	}
	s.updateRepo = repo
	_ = saveShellConfig(shellConfig{PanelURL: s.panelURL, ExternalAPIKey: s.apiKey, ProxyURL: s.proxyURL, UpdateRepo: repo})

	info, err := checkGitHubLatestRelease(context.Background(), repo, s.proxyURL)
	if err != nil {
		return map[string]string{"ok": "false", "msg": err.Error()}, nil
	}
	if !isReleaseNewer(releaseVersion(info), appVersion) {
		return map[string]string{"ok": "false", "msg": "Установлена актуальная версия."}, nil
	}
	exePath, workDir, assetName, err := prepareUpdateExecutable(context.Background(), info, s.proxyURL)
	if err != nil {
		return map[string]string{"ok": "false", "msg": err.Error()}, nil
	}
	if err := launchUpdateInstallerV2(exePath, workDir); err != nil {
		return map[string]string{"ok": "false", "msg": err.Error()}, nil
	}
	s.exitForUpdate()
	return map[string]string{
		"ok":      "true",
		"version": info.TagName,
		"asset":   assetName,
		"msg":     "Обновление скачано. Приложение перезапустится автоматически.",
	}, nil
}

func (s *webShell) exitForUpdate() {
	go func() {
		time.Sleep(700 * time.Millisecond)
		if s.window != nil {
			s.window.PrepareForExit()
		}
		if s.local != nil && s.local.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
			_ = s.local.server.Shutdown(ctx)
			cancel()
		}
		if s.view != nil {
			s.view.Dispatch(func() {
				s.view.Terminate()
			})
		}
		time.Sleep(1800 * time.Millisecond)
		os.Exit(0)
	}()
}

func checkGitHubLatestRelease(ctx context.Context, repo string, proxyURL string) (githubReleaseInfo, error) {
	repo = normalizeUpdateRepo(repo)
	if repo == "" {
		repo = defaultUpdateRepo
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return githubReleaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CMoreeRemotePanel/"+appVersion)

	client := http.Client{Timeout: 13 * time.Second, Transport: newProxyTransport(proxyURL)}
	resp, err := client.Do(req)
	if err != nil {
		return githubReleaseInfo{}, fmt.Errorf("не удалось проверить GitHub Releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return githubReleaseInfo{}, fmt.Errorf("GitHub Release не найден для %s", repo)
	}
	if resp.StatusCode >= 400 {
		return githubReleaseInfo{}, fmt.Errorf("GitHub API ответил HTTP %d", resp.StatusCode)
	}

	var out githubReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return githubReleaseInfo{}, err
	}
	if strings.TrimSpace(out.TagName) == "" {
		return githubReleaseInfo{}, fmt.Errorf("latest release не содержит tag_name")
	}
	return out, nil
}

func releaseResultMap(info githubReleaseInfo) map[string]string {
	asset := selectUpdateAsset(info)
	parsedVersion := releaseVersion(info)
	update := isReleaseNewer(parsedVersion, appVersion)
	body := strings.TrimSpace(info.Body)
	if body == "" {
		body = "Описание релиза не заполнено."
	}
	msg := fmt.Sprintf("Текущая версия %s, latest release %s", appVersion, info.TagName)
	if parsedVersion == "" {
		msg = fmt.Sprintf("Latest release %s не содержит номер версии, обновление не требуется.", info.TagName)
	}
	result := map[string]string{
		"ok":          "true",
		"update":      boolString(update),
		"version":     info.TagName,
		"app_version": parsedVersion,
		"name":        strings.TrimSpace(info.Name),
		"url":         info.HTMLURL,
		"description": body,
		"current":     appVersion,
		"msg":         msg,
	}
	if asset.Name != "" {
		result["asset"] = asset.Name
		result["asset_url"] = asset.BrowserDownloadURL
	}
	return result
}

func selectUpdateAsset(info githubReleaseInfo) githubReleaseAsset {
	var zipAsset githubReleaseAsset
	var exeAsset githubReleaseAsset
	for _, asset := range info.Assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		url := strings.TrimSpace(asset.BrowserDownloadURL)
		if name == "" || url == "" {
			continue
		}
		if strings.Contains(name, "cmoreeremotepanel_portable") && strings.HasSuffix(name, ".zip") {
			return asset
		}
		if zipAsset.Name == "" && strings.HasSuffix(name, ".zip") {
			zipAsset = asset
		}
		if exeAsset.Name == "" && strings.HasSuffix(name, ".exe") {
			exeAsset = asset
		}
	}
	if zipAsset.Name != "" {
		return zipAsset
	}
	return exeAsset
}

func prepareUpdateExecutable(ctx context.Context, info githubReleaseInfo, proxyURL string) (string, string, string, error) {
	asset := selectUpdateAsset(info)
	if asset.Name == "" || asset.BrowserDownloadURL == "" {
		return "", "", "", fmt.Errorf("в релизе %s нет подходящего ассета: нужен CMoreeRemotePanel_portable.zip или .exe", info.TagName)
	}

	workDir := filepath.Join(os.TempDir(), "CMoreeRemotePanelUpdate-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", "", "", err
	}
	assetPath := filepath.Join(workDir, safeUpdateFileName(asset.Name))
	if err := downloadUpdateAsset(ctx, asset.BrowserDownloadURL, proxyURL, assetPath); err != nil {
		return "", "", "", err
	}

	lowerName := strings.ToLower(asset.Name)
	if strings.HasSuffix(lowerName, ".exe") {
		return assetPath, workDir, asset.Name, nil
	}
	if strings.HasSuffix(lowerName, ".zip") {
		exePath, err := extractUpdateExecutable(assetPath, filepath.Join(workDir, "extracted"))
		if err != nil {
			return "", "", "", err
		}
		return exePath, workDir, asset.Name, nil
	}
	return "", "", "", fmt.Errorf("неподдерживаемый тип ассета %s", asset.Name)
}

func downloadUpdateAsset(ctx context.Context, rawURL, proxyURL, dstPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CMoreeRemotePanel/"+appVersion)
	client := http.Client{Timeout: 5 * time.Minute, Transport: newProxyTransport(proxyURL)}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("не удалось скачать обновление: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("GitHub asset ответил HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	tmpPath := dstPath + ".part"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func extractUpdateExecutable(zipPath, dstDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть zip обновления: %w", err)
	}
	defer reader.Close()

	dstAbs, err := filepath.Abs(dstDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dstAbs, 0o755); err != nil {
		return "", err
	}

	var exePath string
	for _, file := range reader.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		cleanName := filepath.Clean(name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}
		target := filepath.Join(dstAbs, cleanName)
		targetAbs, err := filepath.Abs(target)
		if err != nil || !strings.HasPrefix(targetAbs, dstAbs+string(os.PathSeparator)) && targetAbs != dstAbs {
			continue
		}
		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(targetAbs, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return "", err
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		dst, err := os.Create(targetAbs)
		if err != nil {
			_ = src.Close()
			return "", err
		}
		_, copyErr := io.Copy(dst, src)
		closeDstErr := dst.Close()
		closeSrcErr := src.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeDstErr != nil {
			return "", closeDstErr
		}
		if closeSrcErr != nil {
			return "", closeSrcErr
		}
		if strings.EqualFold(filepath.Base(targetAbs), "CMoreeRemotePanel.exe") {
			exePath = targetAbs
		}
	}
	if exePath == "" {
		return "", fmt.Errorf("в zip обновления не найден CMoreeRemotePanel.exe")
	}
	return exePath, nil
}

func launchUpdateInstallerV2(newExePath, workDir string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, _ = filepath.Abs(currentExe)
	newExePath, _ = filepath.Abs(newExePath)
	workDir, _ = filepath.Abs(workDir)

	stamp := time.Now().Format("20060102150405")
	scriptPath := filepath.Join(os.TempDir(), "cmoree-update-"+stamp+".ps1")
	logPath := filepath.Join(os.TempDir(), "cmoree-update-"+stamp+".log")
	script := updateInstallerScript(os.Getpid(), newExePath, currentExe, workDir, logPath)
	if err := writePowerShellScript(scriptPath, script); err != nil {
		return err
	}
	cmd := exec.Command(powershellExePath(), "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windowsCreateNoWindow}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("не удалось запустить updater helper: %w", err)
	}
	return nil
}

func updateInstallerScript(pid int, source, target, workDir, logPath string) string {
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$TargetPid = %d
$Source = %s
$Target = %s
$WorkDir = %s
$Log = %s
$Backup = ''
$Pending = ''
function Write-UpdateLog([string]$Text) {
  try {
    Add-Content -LiteralPath $Log -Encoding UTF8 -Value ("[{0}] {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Text)
  } catch {}
}
try {
  Write-UpdateLog "update helper started"
  Write-UpdateLog ("source=" + $Source)
  Write-UpdateLog ("target=" + $Target)
  $Deadline = (Get-Date).AddSeconds(20)
  while ((Get-Date) -lt $Deadline) {
    $Process = Get-Process -Id $TargetPid -ErrorAction SilentlyContinue
    if ($null -eq $Process) { break }
    Start-Sleep -Milliseconds 250
  }
  $Process = Get-Process -Id $TargetPid -ErrorAction SilentlyContinue
  if ($null -ne $Process) {
    Write-UpdateLog "target still running; requesting close"
    try { $Process.CloseMainWindow() | Out-Null } catch {}
    Start-Sleep -Seconds 2
  }
  $Process = Get-Process -Id $TargetPid -ErrorAction SilentlyContinue
  if ($null -ne $Process) {
    Write-UpdateLog "target still running; force stopping"
    Stop-Process -Id $TargetPid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 1
  }
  if (-not (Test-Path -LiteralPath $Source)) {
    throw ("Downloaded EXE is missing: " + $Source)
  }
  $TargetDir = Split-Path -Parent $Target
  if (-not (Test-Path -LiteralPath $TargetDir)) {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
  }
  $Pending = Join-Path $TargetDir 'CMoreeRemotePanel.exe.pending'
  $Backup = Join-Path $TargetDir ('CMoreeRemotePanel.exe.bak-' + (Get-Date -Format 'yyyyMMddHHmmss'))
  Remove-Item -LiteralPath $Pending -Force -ErrorAction SilentlyContinue
  Copy-Item -LiteralPath $Source -Destination $Pending -Force
  $Installed = $false
  for ($i = 0; $i -lt 40; $i++) {
    try {
      if (Test-Path -LiteralPath $Target) {
        Move-Item -LiteralPath $Target -Destination $Backup -Force
      }
      Move-Item -LiteralPath $Pending -Destination $Target -Force
      $Installed = $true
      break
    } catch {
      Write-UpdateLog ("copy attempt " + $i + " failed: " + $_.Exception.Message)
      if ((Test-Path -LiteralPath $Backup) -and (-not (Test-Path -LiteralPath $Target)) -and (-not (Test-Path -LiteralPath $Pending))) {
        try { Copy-Item -LiteralPath $Backup -Destination $Pending -Force } catch {}
      }
      Start-Sleep -Milliseconds 500
    }
  }
  if (-not $Installed) {
    throw "Could not replace CMoreeRemotePanel.exe"
  }
  Write-UpdateLog "replacement completed"
  try {
    $Started = Start-Process -FilePath $Target -PassThru
    Write-UpdateLog ("started pid=" + $Started.Id)
  } catch {
    Write-UpdateLog ("restart failed: " + $_.Exception.Message)
    if ((Test-Path -LiteralPath $Backup) -and (-not (Test-Path -LiteralPath $Target))) {
      Move-Item -LiteralPath $Backup -Destination $Target -Force
    }
    throw
  }
  Start-Sleep -Seconds 2
  Remove-Item -LiteralPath $Backup -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
  Write-UpdateLog "update finished"
  Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
} catch {
  Write-UpdateLog ("FAILED: " + $_.Exception.Message)
  try {
    if ((Test-Path -LiteralPath $Backup) -and (-not (Test-Path -LiteralPath $Target))) {
      Move-Item -LiteralPath $Backup -Destination $Target -Force
    }
  } catch {}
}
`, pid, psQuote(source), psQuote(target), psQuote(workDir), psQuote(logPath))
}

func writePowerShellScript(path, script string) error {
	encoded := utf16.Encode([]rune(script))
	raw := make([]byte, 2, 2+len(encoded)*2)
	raw[0] = 0xff
	raw[1] = 0xfe
	for _, item := range encoded {
		raw = append(raw, byte(item), byte(item>>8))
	}
	return os.WriteFile(path, raw, 0o600)
}

func powershellExePath() string {
	if root := strings.TrimSpace(os.Getenv("SystemRoot")); root != "" {
		candidate := filepath.Join(root, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "powershell.exe"
}

func launchUpdateInstaller(newExePath, workDir string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, _ = filepath.Abs(currentExe)
	newExePath, _ = filepath.Abs(newExePath)
	workDir, _ = filepath.Abs(workDir)

	scriptPath := filepath.Join(os.TempDir(), "cmoree-update-"+time.Now().Format("20060102150405")+".ps1")
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$TargetPid = %d
$Source = %s
$Target = %s
$WorkDir = %s
$Deadline = (Get-Date).AddSeconds(90)
while ((Get-Date) -lt $Deadline) {
  $Process = Get-Process -Id $TargetPid -ErrorAction SilentlyContinue
  if ($null -eq $Process) { break }
  Start-Sleep -Milliseconds 250
}
Start-Sleep -Milliseconds 700
$Copied = $false
for ($i = 0; $i -lt 40; $i++) {
  try {
    Copy-Item -LiteralPath $Source -Destination $Target -Force
    $Copied = $true
    break
  } catch {
    Start-Sleep -Milliseconds 500
  }
}
if (-not $Copied) { throw 'Не удалось заменить CMoreeRemotePanel.exe' }
Start-Process -FilePath $Target
Start-Sleep -Seconds 2
try { Remove-Item -LiteralPath $WorkDir -Recurse -Force -ErrorAction SilentlyContinue } catch {}
try { Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue } catch {}
`, os.Getpid(), psQuote(newExePath), psQuote(currentExe), psQuote(workDir))

	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func safeUpdateFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	name = re.ReplaceAllString(name, "_")
	if name == "" || name == "." {
		return "CMoreeRemotePanel_update.bin"
	}
	return name
}

func psQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func isReleaseNewer(latestTag, currentVersion string) bool {
	latest := extractReleaseVersion(latestTag)
	current := extractReleaseVersion(currentVersion)
	if latest == "" {
		return false
	}
	if current == "" {
		return false
	}
	if latest == current {
		return false
	}
	latestParts, latestOK := parseVersionParts(latest)
	currentParts, currentOK := parseVersionParts(current)
	if !latestOK || !currentOK {
		return false
	}
	for i := 0; i < len(latestParts) || i < len(currentParts); i++ {
		var l, c int
		if i < len(latestParts) {
			l = latestParts[i]
		}
		if i < len(currentParts) {
			c = currentParts[i]
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return false
}

func releaseVersion(info githubReleaseInfo) string {
	for _, value := range []string{info.TagName, info.Name} {
		if version := extractReleaseVersion(value); version != "" {
			return version
		}
	}
	return ""
}

func extractReleaseVersion(value string) string {
	match := releaseVersionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) < 4 {
		return ""
	}
	return match[1] + "." + match[2] + "." + match[3]
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	return value
}

func parseVersionParts(value string) ([]int, bool) {
	value = normalizeVersion(value)
	if value == "" {
		return nil, false
	}
	rawParts := strings.Split(value, ".")
	parts := make([]int, 0, len(rawParts))
	for _, raw := range rawParts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, false
		}
		digits := strings.Builder{}
		for _, r := range raw {
			if r < '0' || r > '9' {
				break
			}
			digits.WriteRune(r)
		}
		if digits.Len() == 0 {
			return nil, false
		}
		n, err := strconv.Atoi(digits.String())
		if err != nil {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, true
}
