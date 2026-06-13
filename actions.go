//go:build nativeui

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/sqweek/dialog"
)

func (a *DesktopApp) refreshActivePage() {
	if a.mode != "app" {
		a.bootstrapAsync()
		return
	}
	a.loadPage(a.activePage)
}

func (a *DesktopApp) loadPage(id string) {
	page := a.pages[id]
	if page == nil {
		return
	}
	a.runAsync("Обновление "+pageByID(id).Label, func() error {
		data, err := a.fetchPageData(id, page)
		if err != nil {
			return err
		}
		page.SetData(data)
		if st, ok := data["status"].(StatusResponse); ok {
			a.status = st
		}
		return nil
	})
}

func (a *DesktopApp) fetchPageData(id string, p *PageRuntime) (map[string]any, error) {
	out := map[string]any{}
	switch id {
	case "dashboard":
		var status StatusResponse
		if err := a.api.Get(a.ctx, "/api/status", nil, &status); err != nil {
			return nil, err
		}
		a.status = status
		var audit map[string]any
		_ = a.api.Get(a.ctx, "/api/admin-audit", map[string]string{"limit": "8"}, &audit)
		var history map[string]any
		_ = a.api.Get(a.ctx, "/api/client/console/history", nil, &history)
		out["status"] = status
		out["audit"] = audit
		out["console_history"] = history
	case "console":
		_ = a.api.Get(a.ctx, "/api/client/console/history", nil, &out)
		var catalog map[string]any
		_ = a.api.Get(a.ctx, "/api/console/catalog", map[string]string{"limit": "14", "q": p.Search.Text()}, &catalog)
		out["catalog"] = catalog
	case "kits":
		_ = a.api.Get(a.ctx, "/api/kits", nil, &out)
		var catalog map[string]any
		_ = a.api.Get(a.ctx, "/api/kits/catalog", map[string]string{"limit": "16", "q": p.Search.Text()}, &catalog)
		out["catalog"] = catalog
	case "configs":
		_ = a.api.Get(a.ctx, "/api/configs", nil, &out)
		var panel map[string]any
		_ = a.api.Get(a.ctx, "/api/panel_config", nil, &panel)
		out["panel_config"] = panel
	case "files":
		_ = a.api.Get(a.ctx, "/api/file-manager/roots", nil, &out)
		if p.Primary.Text() != "" {
			var listing map[string]any
			_ = a.api.Get(a.ctx, "/api/file-manager/list", map[string]string{"path": p.Primary.Text()}, &listing)
			out["listing"] = listing
		}
	case "addons":
		_ = a.api.Get(a.ctx, "/api/mods", nil, &out)
		var progress map[string]any
		_ = a.api.Get(a.ctx, "/api/addons/download/status", nil, &progress)
		var auto map[string]any
		_ = a.api.Get(a.ctx, "/api/addons/autoupdate", nil, &auto)
		out["download"] = progress
		out["autoupdate"] = auto
	case "logs":
		_ = a.api.Get(a.ctx, "/api/logs", nil, &out)
		if p.Primary.Text() != "" {
			var tail map[string]any
			_ = a.api.Get(a.ctx, "/api/logs/read", map[string]string{"path": p.Primary.Text(), "lines": "1000"}, &tail)
			out["tail"] = tail
		}
	case "database":
		_ = a.api.Get(a.ctx, "/api/dbs", nil, &out)
		if p.Primary.Text() != "" && p.Secondary.Text() != "" {
			var rows map[string]any
			_ = a.api.Get(a.ctx, "/api/db/rows", map[string]string{"path": p.Primary.Text(), "table": p.Secondary.Text(), "search": p.Search.Text()}, &rows)
			out["rows"] = rows
		} else if p.Primary.Text() != "" {
			var tables map[string]any
			_ = a.api.Get(a.ctx, "/api/db/tables", map[string]string{"path": p.Primary.Text()}, &tables)
			out["tables"] = tables
		}
	case "backups":
		_ = a.api.Get(a.ctx, "/api/backups", nil, &out)
	case "discord":
		_ = a.api.Get(a.ctx, "/api/discord", nil, &out)
	case "scheduler":
		_ = a.api.Get(a.ctx, "/api/schedules", nil, &out)
	case "accounts":
		_ = a.api.Get(a.ctx, "/api/users", nil, &out)
	case "audit":
		_ = a.api.Get(a.ctx, "/api/admin-audit", map[string]string{"limit": "200"}, &out)
	}
	return out, nil
}

func (a *DesktopApp) pageAction(id, action string) {
	p := a.pages[id]
	if p == nil {
		return
	}
	switch id {
	case "dashboard":
		a.dashboardAction(action, p)
	case "console":
		a.consoleAction(action, p)
	case "kits":
		a.kitsAction(action, p)
	case "configs":
		a.configAction(action, p)
	case "files":
		a.filesAction(action, p)
	case "addons":
		a.addonsAction(action, p)
	case "logs":
		a.logsAction(action, p)
	case "database":
		a.dbAction(action, p)
	case "backups":
		a.backupsAction(action, p)
	case "discord":
		a.discordAction(action, p)
	case "scheduler":
		a.schedulerAction(action, p)
	case "accounts":
		a.accountsAction(action, p)
	case "audit":
		a.loadPage(id)
	}
}

func (a *DesktopApp) dashboardAction(action string, p *PageRuntime) {
	if action == "refresh" {
		a.loadPage("dashboard")
		return
	}
	if action == "wipe" && strings.TrimSpace(p.Action.Text()) != "WIPE" {
		a.setToast("err", "Для вайпа введите WIPE в поле подтверждения")
		return
	}
	endpoint := map[string]string{
		"start":   "/api/server/start",
		"stop":    "/api/server/stop",
		"restart": "/api/server/restart",
		"wipe":    "/api/server/wipe",
	}[action]
	if endpoint == "" {
		return
	}
	a.runAsync("Действие сервера", func() error {
		var out map[string]any
		if err := a.api.PostJSON(a.ctx, endpoint, map[string]any{"confirm": true}, &out); err != nil {
			return err
		}
		p.SetData(map[string]any{"result": out})
		a.loadPage("dashboard")
		return nil
	})
}

func (a *DesktopApp) consoleAction(action string, p *PageRuntime) {
	switch action {
	case "send":
		command := strings.TrimSpace(p.Primary.Text())
		if command == "" {
			return
		}
		a.runAsync("Команда консоли", func() error {
			var out map[string]any
			if err := a.api.PostJSON(a.ctx, "/api/console", map[string]any{"command": command}, &out); err != nil {
				return err
			}
			p.Primary.SetText("")
			a.loadPage("console")
			return nil
		})
	case "refresh", "search":
		a.loadPage("console")
	}
}

func (a *DesktopApp) kitsAction(action string, p *PageRuntime) {
	switch action {
	case "search", "refresh":
		a.loadPage("kits")
	case "issue":
		id, login := strings.TrimSpace(p.Primary.Text()), strings.TrimSpace(p.Secondary.Text())
		if id == "" || login == "" {
			a.setToast("err", "Укажите kit id и игрока")
			return
		}
		a.runAsync("Выдача кита", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/kits/"+id+"/issue", map[string]any{"login": login}, &out)
		})
	case "save":
		payload, err := parseJSONBody(p.Body.Text())
		if err != nil {
			a.setToast("err", err.Error())
			return
		}
		a.runAsync("Сохранение кита", func() error {
			var out map[string]any
			if err := a.api.PostJSON(a.ctx, "/api/kits", payload, &out); err != nil {
				return err
			}
			a.loadPage("kits")
			return nil
		})
	case "delete":
		id := strings.TrimSpace(p.Primary.Text())
		if id == "" || strings.TrimSpace(p.Action.Text()) != "DELETE" {
			a.setToast("err", "Укажите kit id и DELETE")
			return
		}
		a.runAsync("Удаление кита", func() error {
			var out map[string]any
			if err := a.api.Delete(a.ctx, "/api/kits/"+id, &out); err != nil {
				return err
			}
			a.loadPage("kits")
			return nil
		})
	}
}

func (a *DesktopApp) configAction(action string, p *PageRuntime) {
	switch action {
	case "refresh":
		a.loadPage("configs")
	case "read":
		path := strings.TrimSpace(p.Primary.Text())
		a.runAsync("Чтение конфига", func() error {
			var out map[string]any
			if err := a.api.Get(a.ctx, "/api/config/raw", map[string]string{"path": path}, &out); err != nil {
				return err
			}
			if content := stringValue(out["content"]); content != "" {
				p.Body.SetText(content)
			}
			p.SetData(out)
			return nil
		})
	case "write":
		path := strings.TrimSpace(p.Primary.Text())
		a.runAsync("Сохранение конфига", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/config/raw", map[string]any{"path": path, "content": p.Body.Text()}, &out)
		})
	case "save":
		payload, err := parseJSONBody(p.Body.Text())
		if err != nil {
			a.setToast("err", err.Error())
			return
		}
		a.runAsync("Сохранение panel_config", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/panel_config", payload, &out)
		})
	}
}

func (a *DesktopApp) filesAction(action string, p *PageRuntime) {
	switch action {
	case "refresh", "read":
		a.loadPage("files")
		if action == "read" && p.Primary.Text() != "" {
			a.runAsync("Чтение файла", func() error {
				var out map[string]any
				if err := a.api.Get(a.ctx, "/api/file-manager/read", map[string]string{"path": p.Primary.Text()}, &out); err != nil {
					return err
				}
				p.Body.SetText(stringValue(out["content"]))
				p.SetData(out)
				return nil
			})
		}
	case "write":
		a.runAsync("Запись файла", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/file-manager/write", map[string]any{"path": p.Primary.Text(), "content": p.Body.Text(), "encoding": "utf-8"}, &out)
		})
	case "upload":
		file, err := dialog.File().Title("Выберите файл для загрузки").Load()
		if err != nil {
			return
		}
		a.runAsync("Загрузка файла", func() error {
			var out map[string]any
			return a.api.UploadFile(a.ctx, p.Primary.Text(), file, &out)
		})
	case "save":
		a.runAsync("Создание папки", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/file-manager/mkdir", map[string]any{"path": p.Primary.Text(), "name": p.Secondary.Text()}, &out)
		})
	case "delete":
		if strings.TrimSpace(p.Action.Text()) != "DELETE" {
			a.setToast("err", "Для удаления введите DELETE")
			return
		}
		a.runAsync("Удаление файла", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/file-manager/delete", map[string]any{"path": p.Primary.Text()}, &out)
		})
	}
}

func (a *DesktopApp) addonsAction(action string, p *PageRuntime) {
	switch action {
	case "refresh":
		a.loadPage("addons")
	case "search":
		query := strings.TrimSpace(p.Search.Text())
		a.runAsync("Поиск Steam", func() error {
			var out map[string]any
			if err := a.api.Get(a.ctx, "/api/steam/search", map[string]string{"q": query}, &out); err != nil {
				return err
			}
			p.SetData(out)
			return nil
		})
	case "read":
		id := strings.TrimSpace(p.Primary.Text())
		a.runAsync("Информация Workshop", func() error {
			var out map[string]any
			if err := a.api.Get(a.ctx, "/api/steam/modinfo", map[string]string{"id": id}, &out); err != nil {
				return err
			}
			p.SetData(out)
			return nil
		})
	case "save":
		payload, err := parseJSONBody(p.Body.Text())
		if err != nil {
			a.setToast("err", err.Error())
			return
		}
		a.runAsync("Сохранение аддонов", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/mods", payload, &out)
		})
	case "run":
		ids := splitIDs(p.Primary.Text())
		a.runAsync("Загрузка аддонов", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/addons/download", map[string]any{"workshop": ids}, &out)
		})
	}
}

func (a *DesktopApp) logsAction(action string, p *PageRuntime) {
	a.loadPage("logs")
}

func (a *DesktopApp) dbAction(action string, p *PageRuntime) {
	switch action {
	case "refresh", "read":
		a.loadPage("database")
	case "snapshot":
		a.runAsync("Snapshot БД", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/db/snapshot", map[string]any{"path": p.Primary.Text()}, &out)
		})
	case "diff":
		a.runAsync("Diff БД", func() error {
			var out map[string]any
			if err := a.api.Get(a.ctx, "/api/db/diff", map[string]string{"path": p.Primary.Text()}, &out); err != nil {
				return err
			}
			p.SetData(out)
			return nil
		})
	}
}

func (a *DesktopApp) backupsAction(action string, p *PageRuntime) {
	switch action {
	case "refresh":
		a.loadPage("backups")
	case "upload":
		file, err := dialog.File().Title("Выберите ZIP-архив мира").Load()
		if err != nil {
			return
		}
		if !strings.HasSuffix(strings.ToLower(file), ".zip") {
			a.setToast("err", "Поддерживаются только ZIP-архивы")
			return
		}
		a.runAsync("Загрузка бэкапа", func() error {
			var out map[string]any
			if err := a.api.UploadBackup(a.ctx, file, &out); err != nil {
				return err
			}
			p.SetData(out)
			a.loadPage("backups")
			return nil
		})
	case "restore":
		name := strings.TrimSpace(p.Primary.Text())
		if name == "" || strings.TrimSpace(p.Action.Text()) != "RESTORE" {
			a.setToast("err", "Выберите бэкап и введите RESTORE")
			return
		}
		a.runAsync("Откат бэкапа", func() error {
			var out map[string]any
			if err := a.api.PostJSON(a.ctx, "/api/backups/"+url.PathEscape(name)+"/restore", map[string]any{}, &out); err != nil {
				return err
			}
			p.SetData(out)
			a.loadPage("backups")
			return nil
		})
	case "wipe":
		if strings.TrimSpace(p.Action.Text()) != "WIPE" {
			a.setToast("err", "Для вайпа введите WIPE")
			return
		}
		targets := splitIDs(p.Secondary.Text())
		if len(targets) == 0 {
			targets = []string{"db", "Saves", "Logs"}
		}
		mode := strings.TrimSpace(p.Primary.Text())
		if strings.HasSuffix(strings.ToLower(mode), ".zip") {
			mode = ""
		}
		a.runAsync("Вайп сервера", func() error {
			var out map[string]any
			if err := a.api.PostJSON(a.ctx, "/api/server/wipe", map[string]any{"mode": mode, "targets": targets}, &out); err != nil {
				return err
			}
			p.SetData(out)
			a.loadPage("backups")
			return nil
		})
	}
}

func (a *DesktopApp) discordAction(action string, p *PageRuntime) {
	switch action {
	case "refresh":
		a.loadPage("discord")
	case "test":
		a.runAsync("Тест Discord", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/discord/test", map[string]any{}, &out)
		})
	case "save":
		payload, err := parseJSONBody(p.Body.Text())
		if err != nil {
			a.setToast("err", err.Error())
			return
		}
		a.runAsync("Сохранение Discord", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/discord", payload, &out)
		})
	case "run":
		id := strings.TrimSpace(p.Primary.Text())
		a.runAsync("Запуск Discord-задачи", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/discord/tasks/"+id+"/run", map[string]any{}, &out)
		})
	case "delete":
		id := strings.TrimSpace(p.Primary.Text())
		if strings.TrimSpace(p.Action.Text()) != "DELETE" {
			a.setToast("err", "Для удаления задачи введите DELETE")
			return
		}
		a.runAsync("Удаление Discord-задачи", func() error {
			var out map[string]any
			return a.api.Delete(a.ctx, "/api/discord/tasks/"+id, &out)
		})
	}
}

func (a *DesktopApp) schedulerAction(action string, p *PageRuntime) {
	id := strings.TrimSpace(p.Primary.Text())
	switch action {
	case "refresh":
		a.loadPage("scheduler")
	case "save":
		payload, err := parseJSONBody(p.Body.Text())
		if err != nil {
			a.setToast("err", err.Error())
			return
		}
		a.runAsync("Сохранение расписания", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/schedules", payload, &out)
		})
	case "toggle", "run":
		a.runAsync("Действие расписания", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/schedules/"+id+"/"+action, map[string]any{}, &out)
		})
	case "delete":
		if strings.TrimSpace(p.Action.Text()) != "DELETE" {
			a.setToast("err", "Для удаления расписания введите DELETE")
			return
		}
		a.runAsync("Удаление расписания", func() error {
			var out map[string]any
			return a.api.Delete(a.ctx, "/api/schedules/"+id, &out)
		})
	}
}

func (a *DesktopApp) accountsAction(action string, p *PageRuntime) {
	username := strings.TrimSpace(p.Primary.Text())
	switch action {
	case "refresh":
		a.loadPage("accounts")
	case "save":
		payload, err := parseJSONBody(p.Body.Text())
		if err != nil {
			a.setToast("err", err.Error())
			return
		}
		a.runAsync("Сохранение пользователя", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/users", payload, &out)
		})
	case "toggle":
		a.runAsync("Переключение пользователя", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/users/"+username+"/toggle", map[string]any{}, &out)
		})
	case "read":
		a.runAsync("Просмотр пароля", func() error {
			var out map[string]any
			if err := a.api.Get(a.ctx, "/api/users/"+username+"/secret", nil, &out); err != nil {
				return err
			}
			p.SetData(out)
			return nil
		})
	case "write":
		password := p.Secondary.Text()
		a.runAsync("Смена пароля", func() error {
			var out map[string]any
			return a.api.PostJSON(a.ctx, "/api/users/"+username+"/password", map[string]any{"password": password}, &out)
		})
	case "delete":
		if strings.TrimSpace(p.Action.Text()) != "DELETE" {
			a.setToast("err", "Для удаления пользователя введите DELETE")
			return
		}
		a.runAsync("Удаление пользователя", func() error {
			var out map[string]any
			return a.api.Delete(a.ctx, "/api/users/"+username, &out)
		})
	}
}

func parseJSONBody(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("JSON пустой")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func splitIDs(text string) []string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	})
	out := []string{}
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
