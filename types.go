package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type BootstrapResponse struct {
	OK               bool              `json:"ok"`
	Configured       bool              `json:"configured"`
	Authenticated    bool              `json:"authenticated"`
	User             UserInfo          `json:"user"`
	IsRoot           bool              `json:"is_root"`
	Permissions      []string          `json:"permissions"`
	PermissionGroups []PermissionGroup `json:"permission_groups"`
	RankOptions      []map[string]any  `json:"rank_options"`
	Config           map[string]any    `json:"config"`
	Status           StatusResponse    `json:"status"`
	Nav              map[string]bool   `json:"nav"`
	Msg              string            `json:"msg"`
}

type UserInfo struct {
	Username        string   `json:"username"`
	DisplayName     string   `json:"display_name"`
	Rank            string   `json:"rank"`
	RankLabel       string   `json:"rank_label"`
	IsRoot          bool     `json:"is_root"`
	Enabled         bool     `json:"enabled"`
	Permissions     []string `json:"permissions"`
	PermissionCount int      `json:"permission_count"`
	LastLogin       string   `json:"last_login"`
	LastIP          string   `json:"last_ip"`
}

type PermissionGroup struct {
	ID    string           `json:"id"`
	Label string           `json:"label"`
	Items []map[string]any `json:"items"`
}

type StatusResponse struct {
	Running           bool           `json:"running"`
	PID               any            `json:"pid"`
	MemoryMB          any            `json:"memory_mb"`
	CPU               any            `json:"cpu"`
	ConsoleReady      bool           `json:"console_ready"`
	ProcessMode       string         `json:"process_mode"`
	ProbePending      bool           `json:"probe_pending"`
	ShutdownRequested bool           `json:"shutdown_requested"`
	Server            ServerInfo     `json:"server"`
	Scheduler         map[string]any `json:"scheduler"`
	AddonAutoupdate   map[string]any `json:"addon_autoupdate"`
	OK                bool           `json:"ok"`
	Msg               string         `json:"msg"`
}

type ServerInfo struct {
	ServerName    string `json:"server_name"`
	PublicName    string `json:"public_name"`
	IP            string `json:"ip"`
	Port          any    `json:"port"`
	PortDisplay   any    `json:"port_display"`
	Map           string `json:"map"`
	Public        any    `json:"public"`
	PlayersOnline any    `json:"players_online"`
	PlayersMax    any    `json:"players_max"`
	PlayersList   []any  `json:"players_list"`
}

type PageDef struct {
	ID       string
	Label    string
	Group    string
	Subtitle string
	Perms    []string
	Icon     []byte
}

var pageDefs = []PageDef{
	{ID: "dashboard", Label: "Панель", Group: "Управление", Subtitle: "Статус сервера, быстрые действия и последние события.", Icon: icons.ActionDashboard},
	{ID: "console", Label: "Консоль", Group: "Управление", Subtitle: "Живая консоль, история и отправка команд.", Perms: []string{"console.view"}, Icon: icons.ActionCode},
	{ID: "kits", Label: "Киты", Group: "Управление", Subtitle: "Создание, выдача и каталог предметов.", Perms: []string{"kits.view"}, Icon: icons.ActionRedeem},
	{ID: "configs", Label: "Файлы конфигов", Group: "Конфиги", Subtitle: "INI/raw/sandbox/spawn и настройки панели.", Perms: []string{"configs.view"}, Icon: icons.ActionSettingsApplications},
	{ID: "files", Label: "Мини-файлы", Group: "Конфиги", Subtitle: "Файловый менеджер разрешенных корней.", Perms: []string{"files.view"}, Icon: icons.FileFolderOpen},
	{ID: "addons", Label: "Аддоны", Group: "Workshop", Subtitle: "Mods, WorkshopItems, поиск Steam и автозагрузка.", Perms: []string{"mods.view"}, Icon: icons.ActionExtension},
	{ID: "logs", Label: "Логи", Group: "Мониторинг", Subtitle: "Группы логов и чтение хвоста файлов.", Perms: []string{"logs.view"}, Icon: icons.ActionSubject},
	{ID: "database", Label: "База данных", Group: "Мониторинг", Subtitle: "SQLite таблицы, строки, snapshots и diff.", Perms: []string{"db.view"}, Icon: icons.DeviceStorage},
	{ID: "backups", Label: "Бэкапы", Group: "Мониторинг", Subtitle: "Откат мира, загрузка ZIP и вайп выбранных папок.", Perms: []string{"server.control"}, Icon: icons.ActionSettingsBackupRestore},
	{ID: "discord", Label: "Discord", Group: "Интеграции", Subtitle: "Настройки бота, задачи и тест подключения.", Perms: []string{"discord.view"}, Icon: icons.CommunicationForum},
	{ID: "scheduler", Label: "Планировщик", Group: "Автоматизация", Subtitle: "Расписания действий сервера.", Perms: []string{"scheduler.view"}, Icon: icons.ActionSchedule},
	{ID: "accounts", Label: "Учетные записи", Group: "Учетные записи", Subtitle: "Пользователи, ранги, права и пароли.", Perms: []string{"accounts.view", "accounts.identity", "accounts.permissions"}, Icon: icons.SocialPeople},
	{ID: "audit", Label: "Лог администрации", Group: "Учетные записи", Subtitle: "Журнал действий администраторов.", Perms: []string{"accounts.audit.view"}, Icon: icons.ActionHistory},
}

type SetupField struct {
	Key   string
	Label string
	Hint  string
}

var setupFieldDefs = []SetupField{
	{Key: "server_path", Label: "Папка сервера", Hint: `C:\pzserver`},
	{Key: "server_name", Label: "Имя сервера", Hint: "pzserver"},
	{Key: "start_script", Label: "Скрипт запуска", Hint: `C:\pzserver\start.bat`},
	{Key: "game_path", Label: "Project Zomboid", Hint: `C:\Program Files (x86)\Steam\steamapps\common\Project Zomboid`},
	{Key: "steamapps_path", Label: "steamapps", Hint: `C:\Program Files (x86)\Steam\steamapps`},
	{Key: "steamcmd_path", Label: "steamcmd", Hint: `C:\steamcmd\steamcmd.exe`},
	{Key: "steam_api_key", Label: "Steam API Key", Hint: "optional"},
	{Key: "file_manager_roots", Label: "Корни мини-файлов", Hint: "пути через ;"},
	{Key: "addon_cleanup_path", Label: "Очистка аддонов", Hint: "optional"},
	{Key: "map_cleanup_path", Label: "Очистка карты", Hint: "optional"},
}

type PageRuntime struct {
	ID        string
	List      layout.List
	Search    widget.Editor
	Primary   widget.Editor
	Secondary widget.Editor
	Body      widget.Editor
	Action    widget.Editor
	Buttons   map[string]*widget.Clickable
	Data      map[string]any
	Text      string
	Version   int
	Applied   int
}

func NewPageRuntime(id string) *PageRuntime {
	p := &PageRuntime{
		ID:      id,
		Buttons: make(map[string]*widget.Clickable),
		Data:    make(map[string]any),
	}
	p.Search.SingleLine = true
	p.Primary.SingleLine = true
	p.Secondary.SingleLine = true
	p.Action.SingleLine = true
	p.List.Axis = layout.Vertical
	p.List.Gap = 8
	p.Body.WrapPolicy = 0
	for _, name := range []string{"refresh", "start", "stop", "restart", "send", "save", "delete", "run", "toggle", "read", "write", "upload", "search", "issue", "test", "snapshot", "diff", "wipe", "restore"} {
		p.Buttons[name] = new(widget.Clickable)
	}
	return p
}

func (p *PageRuntime) SetData(data map[string]any) {
	p.Data = data
	p.Text = pretty(data)
	p.Version++
}

func pretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	case int:
		return fmt.Sprint(x)
	case bool:
		if x {
			return "Да"
		}
		return "Нет"
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}

func mapValue(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func sliceValue(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func titleFor(item any) string {
	m := mapValue(item)
	if m == nil {
		return stringValue(item)
	}
	for _, key := range []string{"summary", "name", "title", "label", "username", "display_name", "item_id", "mod_id", "workshop_id", "path", "file", "table", "action", "id"} {
		if value := strings.TrimSpace(stringValue(m[key])); value != "" {
			return value
		}
	}
	return pretty(m)
}

func subtitleFor(item any) string {
	m := mapValue(item)
	if m == nil {
		return ""
	}
	parts := []string{}
	for _, key := range []string{"category_label", "status", "msg", "type", "rank_label", "actor_display", "timestamp_display", "updated_at", "last_login", "last_run", "next_run", "size_display", "path", "workshop_id", "mod_id", "details_text"} {
		if value := strings.TrimSpace(stringValue(m[key])); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return strings.Join(parts, " · ")
}

func pageByID(id string) PageDef {
	for _, page := range pageDefs {
		if page.ID == id {
			return page
		}
	}
	return pageDefs[0]
}

func groupedPages() []string {
	seen := map[string]bool{}
	groups := []string{}
	for _, page := range pageDefs {
		if !seen[page.Group] {
			seen[page.Group] = true
			groups = append(groups, page.Group)
		}
	}
	return groups
}

func hasAny(perms []string, allowed map[string]bool, root bool) bool {
	if root || len(perms) == 0 {
		return true
	}
	for _, perm := range perms {
		if allowed[perm] {
			return true
		}
	}
	return false
}

func permissionSet(perms []string) map[string]bool {
	out := map[string]bool{}
	for _, perm := range perms {
		out[perm] = true
	}
	return out
}

func loadLogoImage() *widget.Image {
	path := filepath.Join("assets", "logo.png")
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		return nil
	}
	op := paint.NewImageOp(img)
	return &widget.Image{Src: op, Fit: widget.Contain, Position: layout.Center}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
