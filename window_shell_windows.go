package main

import (
	"encoding/json"
	"sync"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

const (
	appTitlebarHeight = 34
	appCaptionButtons = 138

	wsCaption    = 0x00C00000
	wsThickFrame = 0x00040000
	wsSysMenu    = 0x00080000
	wsMinBox     = 0x00020000
	wsMaxBox     = 0x00010000

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpFrameChanged = 0x0020

	swHide     = 0
	swShow     = 5
	swMinimize = 6
	swRestore  = 9
	swMaximize = 3

	wmClose       = 0x0010
	wmDestroy     = 0x0002
	wmNCHitTest   = 0x0084
	wmTrayMessage = 0x8000 + 0x4D
	wmUser        = 0x0400
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmLButtonDbl  = 0x0203
	wmRButtonUp   = 0x0205
	wmContextMenu = 0x007B
	ninSelect     = wmUser
	ninKeySelect  = wmUser + 1

	htCaption = 2

	nimAdd        = 0x00000000
	nimDelete     = 0x00000002
	nimSetVersion = 0x00000004
	nifMessage    = 0x00000001
	nifIcon       = 0x00000002
	nifTip        = 0x00000004
	notifyV4      = 4

	imageIcon      = 1
	lrDefaultSize  = 0x00000040
	lrShared       = 0x00008000
	trayIconID     = 1
	menuOpenID     = 1001
	menuExitID     = 1002
	mfString       = 0x00000000
	mfSeparator    = 0x00000800
	tpmRightButton = 0x00000002
	tpmReturnCmd   = 0x00000100
)

var (
	gwlStylePtr    = ^uintptr(15)
	gwlpWndProcPtr = ^uintptr(3)

	user32                  = windows.NewLazySystemDLL("user32.dll")
	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	procSetWindowLongPtrW   = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW   = user32.NewProc("GetWindowLongPtrW")
	procCallWindowProcW     = user32.NewProc("CallWindowProcW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procShowWindow          = user32.NewProc("ShowWindow")
	procIsZoomed            = user32.NewProc("IsZoomed")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procLoadImageW          = user32.NewProc("LoadImageW")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procPostMessageW        = user32.NewProc("PostMessageW")
)

type appWindowShell struct {
	mu          sync.Mutex
	view        webview2.WebView
	hwnd        uintptr
	oldWndProc  uintptr
	wndProc     uintptr
	trayAdded   bool
	allowExit   bool
	closed      bool
	trayTooltip string
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type point struct {
	X int32
	Y int32
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             windows.Handle
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GUIDItem         windows.GUID
	HBalloonIcon     windows.Handle
}

var (
	appShellsMu sync.Mutex
	appShells   = map[uintptr]*appWindowShell{}
)

func installAppWindowShell(w webview2.WebView) *appWindowShell {
	if w == nil || w.Window() == nil {
		return nil
	}
	s := &appWindowShell{
		view:        w,
		hwnd:        uintptr(w.Window()),
		trayTooltip: appTitle,
	}
	s.installFramelessStyle()
	s.subclass()
	s.addTrayIcon()
	return s
}

func (s *appWindowShell) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	s.removeTrayIcon()
	appShellsMu.Lock()
	delete(appShells, s.hwnd)
	appShellsMu.Unlock()
}

func (s *appWindowShell) Minimize() {
	if s == nil {
		return
	}
	_, _, _ = procShowWindow.Call(s.hwnd, swMinimize)
}

func (s *appWindowShell) ToggleMaximize() {
	if s == nil {
		return
	}
	zoomed, _, _ := procIsZoomed.Call(s.hwnd)
	if zoomed != 0 {
		_, _, _ = procShowWindow.Call(s.hwnd, swRestore)
		return
	}
	_, _, _ = procShowWindow.Call(s.hwnd, swMaximize)
}

func (s *appWindowShell) HideToTray() {
	if s == nil {
		return
	}
	_, _, _ = procShowWindow.Call(s.hwnd, swHide)
}

func (s *appWindowShell) PrepareForExit() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.allowExit = true
	s.mu.Unlock()
	s.removeTrayIcon()
}

func (s *appWindowShell) RestoreFromTray() {
	if s == nil {
		return
	}
	_, _, _ = procShowWindow.Call(s.hwnd, swShow)
	_, _, _ = procShowWindow.Call(s.hwnd, swRestore)
	_, _, _ = procSetForegroundWindow.Call(s.hwnd)
}

func (s *appWindowShell) Exit() {
	if s == nil {
		return
	}
	s.PrepareForExit()
	_, _, _ = procPostMessageW.Call(s.hwnd, wmClose, 0, 0)
}

func (s *appWindowShell) installFramelessStyle() {
	style, _, _ := procGetWindowLongPtrW.Call(s.hwnd, gwlStylePtr)
	style &^= wsCaption
	style |= wsThickFrame | wsSysMenu | wsMinBox | wsMaxBox
	_, _, _ = procSetWindowLongPtrW.Call(s.hwnd, gwlStylePtr, style)
	_, _, _ = procSetWindowPos.Call(s.hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpFrameChanged)
}

func (s *appWindowShell) subclass() {
	s.wndProc = windows.NewCallback(appWndProc)
	old, _, _ := procSetWindowLongPtrW.Call(s.hwnd, gwlpWndProcPtr, s.wndProc)
	s.oldWndProc = old
	appShellsMu.Lock()
	appShells[s.hwnd] = s
	appShellsMu.Unlock()
}

func (s *appWindowShell) addTrayIcon() {
	nid := s.newNotifyIconData()
	_, _, _ = procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	nid.UVersion = notifyV4
	_, _, _ = procShellNotifyIconW.Call(nimSetVersion, uintptr(unsafe.Pointer(&nid)))
	s.trayAdded = true
}

func (s *appWindowShell) removeTrayIcon() {
	s.mu.Lock()
	if !s.trayAdded {
		s.mu.Unlock()
		return
	}
	s.trayAdded = false
	s.mu.Unlock()
	nid := s.newNotifyIconData()
	_, _, _ = procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func (s *appWindowShell) newNotifyIconData() notifyIconData {
	nid := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             windows.Handle(s.hwnd),
		UID:              trayIconID,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: wmTrayMessage,
		HIcon:            windows.Handle(loadAppIcon()),
	}
	copy(nid.SzTip[:], windows.StringToUTF16(s.trayTooltip))
	return nid
}

func loadAppIcon() uintptr {
	var hinstance windows.Handle
	_ = windows.GetModuleHandleEx(0, nil, &hinstance)
	icon, _, _ := procLoadImageW.Call(uintptr(hinstance), 2, imageIcon, 0, 0, lrDefaultSize|lrShared)
	if icon != 0 {
		return icon
	}
	icon, _, _ = procLoadImageW.Call(0, 32512, imageIcon, 0, 0, lrDefaultSize|lrShared)
	return icon
}

func appWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	s := appShellFor(hwnd)
	if s == nil {
		return 0
	}
	switch msg {
	case wmNCHitTest:
		if s.isInCustomTitlebar(lParam) {
			return htCaption
		}
	case wmClose:
		s.mu.Lock()
		allowExit := s.allowExit
		s.mu.Unlock()
		if !allowExit {
			s.HideToTray()
			return 0
		}
	case wmDestroy:
		s.removeTrayIcon()
	case wmTrayMessage:
		s.handleTrayEvent(lParam)
		return 0
	}
	return callOldWndProc(s.oldWndProc, hwnd, msg, wParam, lParam)
}

func appShellFor(hwnd uintptr) *appWindowShell {
	appShellsMu.Lock()
	defer appShellsMu.Unlock()
	return appShells[hwnd]
}

func callOldWndProc(oldWndProc, hwnd, msg, wParam, lParam uintptr) uintptr {
	if oldWndProc == 0 {
		return 0
	}
	ret, _, _ := procCallWindowProcW.Call(oldWndProc, hwnd, msg, wParam, lParam)
	return ret
}

func (s *appWindowShell) isInCustomTitlebar(lParam uintptr) bool {
	var r rect
	ok, _, _ := procGetWindowRect.Call(s.hwnd, uintptr(unsafe.Pointer(&r)))
	if ok == 0 {
		return false
	}
	x := int32(int16(lParam & 0xffff))
	y := int32(int16((lParam >> 16) & 0xffff))
	if y < r.Top || y >= r.Top+appTitlebarHeight {
		return false
	}
	if x >= r.Right-appCaptionButtons {
		return false
	}
	return true
}

func (s *appWindowShell) handleTrayEvent(lParam uintptr) {
	switch trayNotificationCode(lParam) {
	case wmLButtonDown, wmLButtonUp, wmLButtonDbl, ninSelect, ninKeySelect:
		s.RestoreFromTray()
	case wmRButtonUp, wmContextMenu:
		s.showTrayMenu()
	}
}

func trayNotificationCode(lParam uintptr) uintptr {
	code := lParam & 0xffff
	switch code {
	case wmLButtonDown, wmLButtonUp, wmLButtonDbl, wmRButtonUp, wmContextMenu, ninSelect, ninKeySelect:
		return code
	}
	return lParam
}

func (s *appWindowShell) showTrayMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		s.RestoreFromTray()
		return
	}
	defer procDestroyMenu.Call(menu)
	appendMenu(menu, mfString, menuOpenID, "Открыть")
	appendMenu(menu, mfSeparator, 0, "")
	appendMenu(menu, mfString, menuExitID, "Выйти")
	var p point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	_, _, _ = procSetForegroundWindow.Call(s.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmRightButton|tpmReturnCmd, uintptr(p.X), uintptr(p.Y), 0, s.hwnd, 0)
	switch cmd {
	case menuOpenID:
		s.RestoreFromTray()
	case menuExitID:
		s.Exit()
	}
}

func appendMenu(menu uintptr, flags uint32, id uintptr, label string) {
	var labelPtr *uint16
	if label != "" {
		labelPtr, _ = windows.UTF16PtrFromString(label)
	}
	_, _, _ = procAppendMenuW.Call(menu, uintptr(flags), id, uintptr(unsafe.Pointer(labelPtr)))
}

func appChromeInitScript() string {
	logo, _ := json.Marshal(appLogoDataURL())
	title, _ := json.Marshal(appTitle)
	return `
(() => {
 const LOGO = ` + string(logo) + `;
 const TITLE = ` + string(title) + `;
 const STYLE_ID = 'cmoree-app-chrome-style';
 const BAR_ID = 'cmoree-app-chrome';
 function ensureChrome(){
  if (document.getElementById(BAR_ID)) return;
  if (!document.getElementById(STYLE_ID)) {
   const style = document.createElement('style');
   style.id = STYLE_ID;
   style.textContent = ` + "`" + `
    :root{--cmoree-chrome-h:34px}
    html{background:#050607!important}
    body{padding-top:var(--cmoree-chrome-h)!important}
    #cmoree-app-chrome{position:fixed!important;top:0!important;left:0!important;right:0!important;height:var(--cmoree-chrome-h)!important;z-index:2147483647!important;background:#050607!important;border-bottom:1px solid rgba(255,255,255,.08)!important;color:#dbdee8!important;display:grid!important;grid-template-columns:148px minmax(0,1fr)138px!important;align-items:center!important;font:12px/1 "Segoe UI",Arial,sans-serif!important;user-select:none!important}
    #cmoree-app-chrome *{box-sizing:border-box!important}
    #cmoree-app-chrome .cm-left{display:flex!important;align-items:center!important;gap:4px!important;padding-left:10px!important;height:100%!important}
    #cmoree-app-chrome .cm-center{height:100%!important;display:flex!important;align-items:center!important;justify-content:center!important;gap:8px!important;font-weight:700!important;color:#f2f3f5!important}
    #cmoree-app-chrome .cm-center img{width:18px!important;height:18px!important;object-fit:contain!important}
    #cmoree-app-chrome button{height:100%!important;border:0!important;background:transparent!important;color:#aeb3bd!important;font:16px/1 "Segoe UI",Arial,sans-serif!important;padding:0!important;margin:0!important;cursor:default!important}
    #cmoree-app-chrome .cm-nav{width:30px!important;border-radius:0!important}
    #cmoree-app-chrome .cm-nav:hover{background:#14161a!important;color:#fff!important}
    #cmoree-app-chrome .cm-controls{display:grid!important;grid-template-columns:repeat(3,46px)!important;height:100%!important}
    #cmoree-app-chrome .cm-controls button{font-size:13px!important}
    #cmoree-app-chrome .cm-controls button:hover{background:#1f2228!important;color:#fff!important}
    #cmoree-app-chrome .cm-controls .cm-close:hover{background:#e81123!important;color:#fff!important}
   ` + "`" + `;
   document.head.appendChild(style);
  }
  const bar = document.createElement('div');
  bar.id = BAR_ID;
  bar.innerHTML = [
   '<div class="cm-left">',
    '<button class="cm-nav" title="Назад" aria-label="Назад">‹</button>',
    '<button class="cm-nav" title="Вперёд" aria-label="Вперёд">›</button>',
   '</div>',
   '<div class="cm-center"><img alt="" src="' + LOGO + '"><span>' + TITLE + '</span></div>',
   '<div class="cm-controls">',
    '<button title="Свернуть" aria-label="Свернуть" data-cm-action="min">—</button>',
    '<button title="Развернуть" aria-label="Развернуть" data-cm-action="max">□</button>',
    '<button class="cm-close" title="Скрыть в трей" aria-label="Скрыть в трей" data-cm-action="close">×</button>',
   '</div>'
  ].join('');
  document.documentElement.appendChild(bar);
  const nav = bar.querySelectorAll('.cm-nav');
  nav[0].addEventListener('click', () => history.back());
  nav[1].addEventListener('click', () => history.forward());
  bar.querySelector('[data-cm-action="min"]').addEventListener('click', () => window.cmoreeWindowMinimize?.());
  bar.querySelector('[data-cm-action="max"]').addEventListener('click', () => window.cmoreeWindowToggleMaximize?.());
  bar.querySelector('[data-cm-action="close"]').addEventListener('click', () => window.cmoreeWindowClose?.());
 }
 if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', ensureChrome, { once:true });
 else ensureChrome();
})();
`
}
