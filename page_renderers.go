//go:build nativeui

package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type richCard struct {
	Title     string
	Sub       string
	Meta      string
	Tone      color.NRGBA
	Primary   string
	Secondary string
	Search    string
	Body      string
}

func (a *DesktopApp) dashboardHero(gtx layout.Context, p *PageRuntime, status StatusResponse) layout.Dimensions {
	name := status.Server.PublicName
	if strings.TrimSpace(name) == "" {
		name = status.Server.ServerName
	}
	if strings.TrimSpace(name) == "" {
		name = stringValue(a.bootstrap.Config["server_name"])
	}
	state := "Offline"
	if status.Running {
		state = "Online"
	}
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.badgeText(gtx, "SERVER") }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 8) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.labelLines(gtx, emptyDash(name), 24, palette.text, true, 2)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 12, 1) }),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										tone := palette.danger
										if status.Running {
											tone = palette.success
										}
										return a.pill(gtx, state, tone)
									}),
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.labelLines(gtx, joinNonEmpty(processStateText(status), "Публичный: "+stringValue(status.Server.Public), status.ProcessMode), 12, palette.muted, false, 2)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.actionButton(gtx, p, "start", "Запустить", "success")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.actionButton(gtx, p, "restart", "Перезапустить", "warning")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.actionButton(gtx, p, "stop", "Остановить", "danger")
							}),
						)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.statGrid(gtx, []statCard{
					{"Статус", state, "Текущее состояние", status.Running},
					{"Адрес", status.Server.IP, fmt.Sprintf("Порт: %s", stringValue(status.Server.PortDisplay)), false},
					{"Карта", status.Server.Map, "server.ini", false},
					{"Игроки", fmt.Sprintf("%s / %s", stringValue(status.Server.PlayersOnline), stringValue(status.Server.PlayersMax)), "онлайн", false},
					{"PID", stringValue(status.PID), status.ProcessMode, false},
					{"Память", stringValue(status.MemoryMB), "RSS MB", false},
					{"CPU", stringValue(status.CPU), "нагрузка", false},
				})
			}),
		)
	})
}

func (a *DesktopApp) dashboardQuickActions(gtx layout.Context, p *PageRuntime) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.pageJumpButton(gtx, p, "console", "Консоль")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.pageJumpButton(gtx, p, "configs", "Конфиги")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageJumpButton(gtx, p, "files", "Файлы") }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageJumpButton(gtx, p, "addons", "Аддоны") }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageJumpButton(gtx, p, "logs", "Логи") }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageJumpButton(gtx, p, "database", "База") }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.editorBox(gtx, &p.Action, "WIPE для вайпа", true)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.pageJumpButton(gtx, p, "backups", "Откат")
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.actionButton(gtx, p, "wipe", "Вайп", "danger")
			}),
		)
	})
}

func (a *DesktopApp) pageJumpButton(gtx layout.Context, p *PageRuntime, id, label string) layout.Dimensions {
	name := "jump_" + id
	btn := p.Buttons[name]
	if btn == nil {
		btn = new(widget.Clickable)
		p.Buttons[name] = btn
	}
	if btn.Clicked(gtx) {
		a.activePage = id
		a.loadPage(id)
	}
	return a.button(gtx, btn, label, "primary")
}

func (a *DesktopApp) consolePage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.62, func(gtx layout.Context) layout.Dimensions {
					return a.consoleLogPanel(gtx, p, slicePath(p.Data, "items"))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.38, func(gtx layout.Context) layout.Dimensions {
					items := slicePath(p.Data, "catalog", "items")
					return a.cardsPanel(gtx, p, "Каталог команд и предметов", countLabel(items), cardsFromItems("catalog", items, p.Search.Text()), "Введите запрос и нажмите \"Каталог\".")
				}),
			)
		}),
	)
}

func (a *DesktopApp) kitsPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	items := slicePath(p.Data, "items")
	history := slicePath(p.Data, "history")
	catalog := slicePath(p.Data, "catalog", "items")
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.statStrip(gtx, statCardsFromMap(mapPath(p.Data, "summary"), []statCard{
				{"Китов", fmt.Sprint(len(items)), "доступно", false},
				{"История", fmt.Sprint(len(history)), "последних выдач", false},
				{"Каталог", emptyDash(stringValue(mapPath(p.Data, "catalog")["total"])), "предметов", false},
			}))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.42, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Наборы", countLabel(items), cardsFromItems("kit", items, p.Search.Text()), "Киты не найдены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.34, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Каталог предметов", countLabel(catalog), cardsFromItems("catalog", catalog, p.Search.Text()), "Введите поиск и нажмите \"Каталог\".")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.24, func(gtx layout.Context) layout.Dimensions {
					return a.editorPanel(gtx, def, p)
				}),
			)
		}),
	)
}

func (a *DesktopApp) configsPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	return a.editorBackedPage(gtx, def, p, "Файлы конфигурации", cardsFromItems("file", slicePath(p.Data, "files"), p.Search.Text()), "Конфиги не найдены.")
}

func (a *DesktopApp) filesPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	items := slicePath(p.Data, "listing", "items")
	title := "Содержимое папки"
	if len(items) == 0 {
		items = slicePath(p.Data, "roots")
		title = "Доступные корни"
	}
	return a.editorBackedPage(gtx, def, p, title, cardsFromItems("file", items, p.Search.Text()), "Корни или файлы не найдены.")
}

func (a *DesktopApp) addonsPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	workshop := firstNonEmptySlice(p.Data, []string{"workshop", "workshop_items", "items"}, []string{"meta", "workshop"})
	mods := firstNonEmptySlice(p.Data, []string{"mods", "mod_ids"}, []string{"meta", "mods"})
	download := mapPath(p.Data, "download")
	auto := firstNonEmptyMap(p.Data, "autoupdate", "addon_autoupdate")
	status := []statCard{
		{"Workshop", fmt.Sprint(len(workshop)), "WorkshopItems", false},
		{"Mods", fmt.Sprint(len(mods)), "Mod ID", false},
		{"Загрузка", progressText(download), stringValue(download["started_at"]), boolValue(download["running"])},
		{"Автообновление", autoupdateText(auto), stringValue(auto["last_check_at"]), boolValue(auto["enabled"])},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.statStrip(gtx, status) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.40, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "WorkshopItems", countLabel(workshop), cardsFromWorkshop(workshop, mapPath(p.Data, "meta", "workshop_meta"), p.Search.Text()), "WorkshopItems не загружены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.34, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Mods", countLabel(mods), cardsFromMods(mods, mapPath(p.Data, "meta", "mod_sources"), p.Search.Text()), "Mod ID не найдены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.26, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Состояние", "", cardsFromMaps("state", []map[string]any{download, auto}), "Нет данных.")
				}),
			)
		}),
	)
}

func (a *DesktopApp) logsPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	groups := slicePath(p.Data, "groups")
	files := flattenNested(groups, "files")
	lines := slicePath(p.Data, "tail", "lines")
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.28, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Пакеты логов", countLabel(groups), cardsFromItems("log_group", groups, p.Search.Text()), "Логи не найдены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.30, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Файлы", countLabel(files), cardsFromItems("file", files, p.Search.Text()), "Выберите пакет логов.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.42, func(gtx layout.Context) layout.Dimensions {
					return a.linesPanel(gtx, "Хвост файла", lines, "Введите путь и нажмите \"Читать путь\".")
				}),
			)
		}),
	)
}

func (a *DesktopApp) databasePage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	files := slicePath(p.Data, "files")
	tables := slicePath(p.Data, "tables", "tables")
	rows := slicePath(p.Data, "rows", "rows")
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.30, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Файлы DB", countLabel(files), cardsFromItems("file", files, p.Search.Text()), "Базы данных не найдены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.28, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Таблицы", countLabel(tables), cardsFromItems("table", tables, p.Search.Text()), "Введите путь к DB и нажмите \"Таблицы/строки\".")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.42, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Строки", countLabel(rows), cardsFromItems("row", rows, p.Search.Text()), "Выберите таблицу, чтобы загрузить строки.")
				}),
			)
		}),
	)
}

func (a *DesktopApp) backupsPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	items := slicePath(p.Data, "items")
	backupDir := stringValue(p.Data["backup_dir"])
	stats := []statCard{
		{"Архивов", fmt.Sprint(len(items)), "backups/startup", false},
		{"Страховки", fmt.Sprint(countKind(items, "world_upload")), "перед загрузкой мира", false},
		{"Обычные", fmt.Sprint(countKind(items, "startup")), "серверные точки", false},
		{"Каталог", emptyDash(backupDir), "путь бэкапов", false},
	}
	actions := []richCard{
		{Title: "Откат сервера", Sub: "Кликните архив, введите RESTORE в подтверждение и нажмите \"Откатить\".", Meta: "Заменяются Saves и db."},
		{Title: "Загрузка мира", Sub: "Нажмите \"Загрузить ZIP\" и выберите архив мира.", Meta: "Перед заменой сервер создаст страховочный бэкап."},
		{Title: "Вайп", Sub: "Введите WIPE и цели через запятую: db, Saves, Logs, steamapps.", Meta: "Если цели пустые, будет использовано db, Saves, Logs.", Tone: palette.danger},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.statStrip(gtx, stats) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.62, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Откат сервера", countLabel(items), cardsFromItems("backup", items, p.Search.Text()), "Бэкапы не найдены в backups/startup.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.38, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Flexed(.48, func(gtx layout.Context) layout.Dimensions {
							return a.cardsPanel(gtx, p, "Обслуживание мира", "", actions, "Нет действий.")
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
						layout.Flexed(.52, func(gtx layout.Context) layout.Dimensions {
							return a.editorPanel(gtx, def, p)
						}),
					)
				}),
			)
		}),
	)
}

func (a *DesktopApp) discordPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	tasks := slicePath(p.Data, "tasks")
	settings := firstNonEmptyMap(p.Data, "settings", "config")
	enabled := boolValue(p.Data["enabled"]) || boolValue(settings["enabled"])
	token := boolValue(p.Data["token_configured"]) || boolValue(settings["token_configured"])
	stats := []statCard{
		{"Интеграция", yesNo(enabled), "Discord bot", enabled},
		{"Токен", yesNo(token), "без показа секрета", token},
		{"Задачи", fmt.Sprint(len(tasks)), "уведомления", false},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.statStrip(gtx, stats) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.42, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Задачи Discord", countLabel(tasks), cardsFromItems("discord_task", tasks, p.Search.Text()), "Задачи не настроены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.30, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Подключение", "", cardsFromMaps("discord", []map[string]any{settings}), "Настройки не загружены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.28, func(gtx layout.Context) layout.Dimensions { return a.editorPanel(gtx, def, p) }),
			)
		}),
	)
}

func (a *DesktopApp) schedulerPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	items := slicePath(p.Data, "items")
	summary := mapPath(p.Data, "summary")
	stats := []statCard{
		{"Всего", fallback(summary["total"], len(items)), "задач", false},
		{"Активно", stringValue(summary["enabled"]), "включено", true},
		{"К запуску", stringValue(summary["due"]), "готовы сейчас", false},
		{"Ближайший", emptyDash(stringValue(summary["next_run"])), "следующее окно", false},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.statStrip(gtx, stats) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.64, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Расписания", countLabel(items), cardsFromItems("schedule", items, p.Search.Text()), "Расписаний нет.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.36, func(gtx layout.Context) layout.Dimensions { return a.editorPanel(gtx, def, p) }),
			)
		}),
	)
}

func (a *DesktopApp) accountsPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	users := slicePath(p.Data, "users")
	rootCount, enabledCount := 0, 0
	for _, item := range users {
		m := mapValue(item)
		if boolValue(m["is_root"]) {
			rootCount++
		}
		if boolValue(m["enabled"]) {
			enabledCount++
		}
	}
	stats := []statCard{
		{"Всего", fmt.Sprint(len(users)), "учеток", false},
		{"Активны", fmt.Sprint(enabledCount), "включены", true},
		{"Root", fmt.Sprint(rootCount), "владелец", false},
		{"Ранги", fmt.Sprint(len(slicePath(p.Data, "ranks"))), "вариантов", false},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.statStrip(gtx, stats) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.64, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Учетные записи", countLabel(users), cardsFromItems("user", users, p.Search.Text()), "Пользователи не найдены.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.36, func(gtx layout.Context) layout.Dimensions { return a.editorPanel(gtx, def, p) }),
			)
		}),
	)
}

func (a *DesktopApp) auditPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	items := slicePath(p.Data, "items")
	stats := []statCard{
		{"Записей", fallback(p.Data["total"], len(items)), "журнал", false},
		{"Категорий", fmt.Sprint(len(slicePath(p.Data, "categories"))), "типов действий", false},
		{"Авторов", fmt.Sprint(len(slicePath(p.Data, "actors"))), "участников", false},
		{"Ошибки", fmt.Sprint(countStatus(items, "error")), "требуют внимания", countStatus(items, "error") == 0},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.statStrip(gtx, stats) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.cardsPanel(gtx, p, "События администрации", countLabel(items), cardsFromItems("audit", items, p.Search.Text()), "По журналу ничего не найдено.")
		}),
	)
}

func (a *DesktopApp) editorBackedPage(gtx layout.Context, def PageDef, p *PageRuntime, title string, cards []richCard, empty string) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(.45, func(gtx layout.Context) layout.Dimensions {
			return a.cardsPanel(gtx, p, title, fmt.Sprint(len(cards)), cards, empty)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(.55, func(gtx layout.Context) layout.Dimensions { return a.editorPanel(gtx, def, p) }),
	)
}

func (a *DesktopApp) statStrip(gtx layout.Context, cards []statCard) layout.Dimensions {
	return a.statGrid(gtx, cards)
}

func (a *DesktopApp) cardsPanel(gtx layout.Context, p *PageRuntime, title, badge string, cards []richCard, empty string) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.panelHead(gtx, title, badge) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 12) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(cards) == 0 {
					return a.emptyState(gtx, empty)
				}
				return a.richGrid(gtx, p, cards)
			}),
		)
	})
}

func (a *DesktopApp) consoleLogPanel(gtx layout.Context, p *PageRuntime, items []any) layout.Dimensions {
	cards := cardsFromItems("console", items, "")
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.panelHead(gtx, "Поток консоли", countLabel(items))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 12) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(cards) == 0 {
					return a.emptyState(gtx, "Консоль пока пустая. Отправьте команду или обновите поток.")
				}
				return a.richList(gtx, p, cards, 68)
			}),
		)
	})
}

func (a *DesktopApp) linesPanel(gtx layout.Context, title string, lines []any, empty string) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.panelHead(gtx, title, countLabel(lines)) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 12) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(lines) == 0 {
					return a.emptyState(gtx, empty)
				}
				textLines := make([]richCard, 0, len(lines))
				for _, line := range lines {
					textLines = append(textLines, richCard{Title: lineText(line), Tone: palette.muted})
				}
				return a.richList(gtx, nil, textLines, 44)
			}),
		)
	})
}

func (a *DesktopApp) panelHead(gtx layout.Context, title, badge string) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, title, 15, palette.text, true) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if strings.TrimSpace(badge) == "" {
				return layout.Dimensions{}
			}
			return a.badgeText(gtx, badge)
		}),
	)
}

func (a *DesktopApp) badgeText(gtx layout.Context, textValue string) layout.Dimensions {
	return rounded(gtx, rgba(115, 174, 255, 28), 18, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(9), Right: unit.Dp(9)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, textValue, 11, palette.accent, true)
		})
	})
}

func (a *DesktopApp) richGrid(gtx layout.Context, p *PageRuntime, cards []richCard) layout.Dimensions {
	cols := 4
	if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(760)) {
		cols = 1
	} else if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(1100)) {
		cols = 2
	} else if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(1480)) {
		cols = 3
	}
	rows := (len(cards) + cols - 1) / cols
	children := make([]layout.FlexChild, 0, rows)
	for r := 0; r < rows; r++ {
		row := r
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rowChildren := make([]layout.FlexChild, 0, cols)
			for c := 0; c < cols; c++ {
				idx := row*cols + c
				if idx >= len(cards) {
					rowChildren = append(rowChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }))
					continue
				}
				card := cards[idx]
				rowChildren = append(rowChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.selectableRichCard(gtx, p, card, idx, 72)
					})
				}))
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, rowChildren...)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *DesktopApp) richList(gtx layout.Context, p *PageRuntime, cards []richCard, minHeight int) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(cards))
	for i, item := range cards {
		idx := i
		card := item
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.selectableRichCard(gtx, p, card, idx, minHeight)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (a *DesktopApp) selectableRichCard(gtx layout.Context, p *PageRuntime, card richCard, index, minHeight int) layout.Dimensions {
	if p == nil || !card.hasSelection() {
		return a.richCard(gtx, card, minHeight)
	}
	key := fmt.Sprintf("select:%d:%s:%s:%s:%s", index, card.Primary, card.Secondary, card.Search, card.Title)
	btn := p.Buttons[key]
	if btn == nil {
		btn = new(widget.Clickable)
		p.Buttons[key] = btn
	}
	if btn.Clicked(gtx) {
		if card.Primary != "" {
			p.Primary.SetText(card.Primary)
		}
		if card.Secondary != "" {
			p.Secondary.SetText(card.Secondary)
		}
		if card.Search != "" {
			p.Search.SetText(card.Search)
		}
		if card.Body != "" {
			p.Body.SetText(card.Body)
		}
		a.loadPage(p.ID)
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return a.richCard(gtx, card, minHeight)
	})
}

func (card richCard) hasSelection() bool {
	return strings.TrimSpace(card.Primary+card.Secondary+card.Search+card.Body) != ""
}

func (a *DesktopApp) richCard(gtx layout.Context, card richCard, minHeight int) layout.Dimensions {
	return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(minHeight)), func(gtx layout.Context) layout.Dimensions {
		return rounded(gtx, palette.surface2, 12, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				titleColor := palette.text
				if card.Tone.A != 0 {
					titleColor = card.Tone
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.labelLines(gtx, emptyDash(card.Title), 13, titleColor, true, 2)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if strings.TrimSpace(card.Sub) == "" {
							return layout.Dimensions{}
						}
						return a.labelLines(gtx, card.Sub, 11, palette.muted, false, 2)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if strings.TrimSpace(card.Meta) == "" {
							return layout.Dimensions{}
						}
						return a.labelLines(gtx, card.Meta, 10, palette.dim, false, 2)
					}),
				)
			})
		})
	})
}

func (a *DesktopApp) emptyState(gtx layout.Context, textValue string) layout.Dimensions {
	return rounded(gtx, rgba(255, 255, 255, 6), 14, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.labelLines(gtx, textValue, 13, palette.muted, false, 4)
		})
	})
}

func (a *DesktopApp) labelLines(gtx layout.Context, textValue string, size float32, col color.NRGBA, bold bool, maxLines int) layout.Dimensions {
	l := material.Label(a.theme, unit.Sp(size), textValue)
	l.Color = col
	l.MaxLines = maxLines
	l.Alignment = text.Start
	if bold {
		l.Font.Weight = 700
	}
	return l.Layout(gtx)
}

func cardsFromItems(kind string, items []any, query string) []richCard {
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]richCard, 0, len(items))
	for _, item := range items {
		card := cardFromItem(kind, item)
		if query != "" && !strings.Contains(strings.ToLower(card.Title+" "+card.Sub+" "+card.Meta), query) {
			continue
		}
		out = append(out, card)
	}
	return out
}

func cardsFromMaps(kind string, items []map[string]any) []richCard {
	out := []richCard{}
	for _, item := range items {
		if len(item) == 0 {
			continue
		}
		out = append(out, cardFromItem(kind, item))
	}
	return out
}

func cardsFromWorkshop(items []any, meta map[string]any, query string) []richCard {
	query = strings.ToLower(strings.TrimSpace(query))
	out := []richCard{}
	for _, item := range items {
		id := strings.TrimSpace(stringValue(item))
		if id == "" {
			continue
		}
		entry := mapValue(meta[id])
		card := richCard{
			Title:   firstString(entry, "title", "name"),
			Sub:     joinNonEmpty(id, firstString(entry, "kind_label", "kind"), countField(entry, "dependencies", "зависимостей")),
			Meta:    firstString(entry, "warning", "garbage_reason"),
			Tone:    toneForStatus(firstString(entry, "status")),
			Primary: id,
		}
		if card.Title == "" {
			card.Title = id
		}
		if query != "" && !strings.Contains(strings.ToLower(card.Title+" "+card.Sub+" "+card.Meta), query) {
			continue
		}
		out = append(out, card)
	}
	return out
}

func cardsFromMods(items []any, sources map[string]any, query string) []richCard {
	query = strings.ToLower(strings.TrimSpace(query))
	out := []richCard{}
	for _, item := range items {
		id := strings.TrimSpace(stringValue(item))
		if id == "" {
			continue
		}
		source := mapValue(sources[id])
		card := richCard{
			Title:   id,
			Sub:     joinNonEmpty(firstString(source, "title"), firstString(source, "workshop_id"), firstString(source, "source")),
			Meta:    firstString(source, "path", "root"),
			Primary: id,
		}
		if query != "" && !strings.Contains(strings.ToLower(card.Title+" "+card.Sub+" "+card.Meta), query) {
			continue
		}
		out = append(out, card)
	}
	return out
}

func cardFromItem(kind string, item any) richCard {
	m := mapValue(item)
	if m == nil {
		return richCard{Title: stringValue(item), Tone: palette.text}
	}
	switch kind {
	case "audit":
		return richCard{
			Title:   firstString(m, "summary", "action", "id"),
			Sub:     joinNonEmpty(firstString(m, "category_label", "category"), firstString(m, "actor_display", "actor"), firstString(m, "status")),
			Meta:    joinNonEmpty(firstString(m, "timestamp_display", "timestamp"), firstString(m, "target"), firstString(m, "details_text")),
			Tone:    toneForStatus(firstString(m, "status")),
			Primary: firstString(m, "id"),
		}
	case "console":
		command := firstString(m, "command")
		if command == "" {
			command = firstString(m, "line", "message", "text", "msg")
		}
		return richCard{
			Title:   firstString(m, "line", "message", "text", "msg", "command"),
			Sub:     joinNonEmpty(firstString(m, "level", "type", "source"), firstString(m, "timestamp_display", "timestamp", "time")),
			Tone:    toneForStatus(firstString(m, "level", "status")),
			Primary: command,
		}
	case "kit":
		return richCard{
			Title:   firstString(m, "name", "title", "id"),
			Sub:     joinNonEmpty(firstString(m, "description"), countField(m, "items", "предметов"), firstString(m, "updated_at", "created_at")),
			Meta:    firstString(m, "id"),
			Primary: firstString(m, "id"),
			Body:    pretty(m),
		}
	case "catalog":
		itemID := firstString(m, "item_id", "id")
		return richCard{
			Title:  firstString(m, "label_ru", "label", "display_name", "item_id", "name"),
			Sub:    joinNonEmpty(firstString(m, "type"), firstString(m, "item_id"), firstString(m, "source_label", "mod_id")),
			Meta:   joinNonEmpty(firstString(m, "module"), firstString(m, "workshop_id")),
			Search: itemID,
		}
	case "workshop":
		id := firstString(m, "workshop_id", "id")
		return richCard{
			Title:   firstString(m, "title", "workshop_id", "id"),
			Sub:     joinNonEmpty(firstString(m, "kind_label", "kind"), firstString(m, "workshop_id", "id"), countField(m, "dependencies", "зависимостей")),
			Meta:    firstString(m, "warning", "garbage_reason"),
			Tone:    toneForStatus(firstString(m, "status")),
			Primary: id,
		}
	case "mod":
		return richCard{Title: firstString(m, "mod_id", "id", "name", "title"), Sub: joinNonEmpty(firstString(m, "workshop_id"), firstString(m, "path", "root")), Primary: firstString(m, "mod_id", "id", "name")}
	case "discord_task":
		return richCard{
			Title:   firstString(m, "name", "id", "type"),
			Sub:     joinNonEmpty(firstString(m, "type"), enabledText(m), firstString(m, "interval_seconds", "channel_id")),
			Meta:    joinNonEmpty(firstString(m, "last_run_at", "last_run"), firstString(m, "last_error")),
			Tone:    enabledTone(m),
			Primary: firstString(m, "id"),
			Body:    pretty(m),
		}
	case "schedule":
		return richCard{
			Title:   firstString(m, "name", "id"),
			Sub:     joinNonEmpty(firstString(m, "action"), firstString(m, "run_at"), enabledText(m)),
			Meta:    joinNonEmpty(firstString(m, "next_run", "last_run"), firstString(m, "msg")),
			Tone:    enabledTone(m),
			Primary: firstString(m, "id"),
			Body:    pretty(m),
		}
	case "user":
		return richCard{
			Title:   firstString(m, "display_name", "username"),
			Sub:     joinNonEmpty(firstString(m, "username"), firstString(m, "rank_label", "rank"), enabledText(m)),
			Meta:    joinNonEmpty(firstString(m, "last_login"), firstString(m, "last_ip"), countField(m, "permissions", "прав")),
			Tone:    enabledTone(m),
			Primary: firstString(m, "username"),
			Body:    pretty(m),
		}
	case "file":
		return richCard{Title: firstString(m, "name", "file", "label", "path"), Sub: joinNonEmpty(firstString(m, "type"), firstString(m, "size_display", "size"), firstString(m, "path")), Meta: firstString(m, "mtime_display", "modified_display", "updated_at"), Primary: firstString(m, "path")}
	case "log_group":
		return richCard{Title: firstString(m, "label", "name", "folder", "path"), Sub: joinNonEmpty(countLike(m, "file_count", "файлов"), firstString(m, "date_display"), firstString(m, "folder")), Meta: joinNonEmpty(countLike(m, "total_size", "байт"), firstString(m, "path"))}
	case "table":
		return richCard{Title: firstString(m, "name", "table"), Sub: joinNonEmpty(countLike(m, "row_count", "строк"), countLike(m, "rows", "строк"), countField(m, "columns", "колонок"), firstString(m, "size_display")), Meta: firstString(m, "changed", "snapshot_at"), Secondary: firstString(m, "name", "table")}
	case "backup":
		tone := color.NRGBA{}
		if firstString(m, "kind") == "world_upload" {
			tone = palette.accent
		}
		return richCard{
			Title:   firstString(m, "name", "file", "path"),
			Sub:     joinNonEmpty(firstString(m, "kind_label", "kind"), firstString(m, "modified_at_moscow_display", "modified_at_moscow", "modified_at")),
			Meta:    joinNonEmpty(formatBytes(m["size"]), firstString(m, "path")),
			Tone:    tone,
			Primary: firstString(m, "name"),
		}
	case "row":
		return richCard{Title: titleFor(m), Sub: compactMap(m, 3)}
	case "state", "discord":
		return richCard{Title: firstString(m, "name", "status_label", "msg", "last_result", "bot_name", "id"), Sub: compactMap(m, 4), Tone: toneForStatus(firstString(m, "status", "status_kind"))}
	default:
		return richCard{Title: titleFor(m), Sub: subtitleFor(m)}
	}
}

func slicePath(data map[string]any, keys ...string) []any {
	if len(keys) == 0 {
		return nil
	}
	cur := any(data)
	for _, key := range keys {
		m := mapValue(cur)
		if m == nil {
			return nil
		}
		cur = m[key]
	}
	return sliceValue(cur)
}

func mapPath(data map[string]any, keys ...string) map[string]any {
	if len(keys) == 0 {
		return data
	}
	cur := any(data)
	for _, key := range keys {
		m := mapValue(cur)
		if m == nil {
			return map[string]any{}
		}
		cur = m[key]
	}
	if m := mapValue(cur); m != nil {
		return m
	}
	return map[string]any{}
}

func firstNonEmptySlice(data map[string]any, direct []string, nested ...[]string) []any {
	for _, key := range direct {
		if items := sliceValue(data[key]); len(items) > 0 {
			return items
		}
	}
	for _, keys := range nested {
		if items := slicePath(data, keys...); len(items) > 0 {
			return items
		}
	}
	return nil
}

func firstNonEmptyMap(data map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if m := mapValue(data[key]); len(m) > 0 {
			return m
		}
	}
	return map[string]any{}
}

func flattenNested(items []any, key string) []any {
	out := []any{}
	for _, item := range items {
		m := mapValue(item)
		if nested := sliceValue(m[key]); len(nested) > 0 {
			out = append(out, nested...)
		}
	}
	return out
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(stringValue(m[key])); value != "" {
			return value
		}
	}
	return ""
}

func joinNonEmpty(parts ...string) string {
	out := []string{}
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" && part != "-" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " · ")
}

func countLabel(items []any) string {
	return fmt.Sprint(len(items))
}

func countField(m map[string]any, key, label string) string {
	if items := sliceValue(m[key]); len(items) > 0 {
		return fmt.Sprintf("%d %s", len(items), label)
	}
	return ""
}

func countLike(m map[string]any, key, label string) string {
	value := strings.TrimSpace(stringValue(m[key]))
	if value == "" {
		return ""
	}
	return value + " " + label
}

func compactMap(m map[string]any, limit int) string {
	parts := []string{}
	for _, key := range sortedKeys(m) {
		if len(parts) >= limit {
			break
		}
		value := stringValue(m[key])
		if strings.TrimSpace(value) == "" || strings.Contains(key, "password") || strings.Contains(key, "token") {
			continue
		}
		parts = append(parts, key+": "+value)
	}
	return strings.Join(parts, " · ")
}

func lineText(v any) string {
	if m := mapValue(v); m != nil {
		return firstString(m, "line", "message", "text", "msg", "command")
	}
	return stringValue(v)
}

func boolValue(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		x = strings.ToLower(strings.TrimSpace(x))
		return x == "true" || x == "yes" || x == "on" || x == "1" || x == "да" || x == "включено"
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}

func yesNo(v bool) string {
	if v {
		return "Да"
	}
	return "Нет"
}

func enabledText(m map[string]any) string {
	if _, ok := m["enabled"]; ok {
		if boolValue(m["enabled"]) {
			return "Включено"
		}
		return "Отключено"
	}
	return ""
}

func enabledTone(m map[string]any) color.NRGBA {
	if _, ok := m["enabled"]; ok && boolValue(m["enabled"]) {
		return palette.success
	}
	return color.NRGBA{}
}

func toneForStatus(value string) color.NRGBA {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ok", "info", "online", "running", "on":
		return palette.success
	case "warn", "warning":
		return palette.warning
	case "err", "error", "offline", "off", "stopped":
		return palette.danger
	default:
		return color.NRGBA{}
	}
}

func progressText(m map[string]any) string {
	if len(m) == 0 {
		return "-"
	}
	if boolValue(m["running"]) {
		return fmt.Sprintf("%s / %s", stringValue(m["done"]), stringValue(m["total"]))
	}
	if boolValue(m["ok"]) {
		return "Готово"
	}
	return "Ожидание"
}

func autoupdateText(m map[string]any) string {
	if len(m) == 0 {
		return "-"
	}
	if label := stringValue(m["status_label"]); label != "" {
		return label
	}
	return yesNo(boolValue(m["enabled"]))
}

func countStatus(items []any, status string) int {
	total := 0
	for _, item := range items {
		if strings.EqualFold(firstString(mapValue(item), "status"), status) {
			total++
		}
	}
	return total
}

func countKind(items []any, kind string) int {
	total := 0
	for _, item := range items {
		if strings.EqualFold(firstString(mapValue(item), "kind"), kind) {
			total++
		}
	}
	return total
}

func fallback(value any, alt int) string {
	if text := strings.TrimSpace(stringValue(value)); text != "" {
		return text
	}
	return fmt.Sprint(alt)
}

func formatBytes(value any) string {
	var size float64
	switch x := value.(type) {
	case float64:
		size = x
	case int:
		size = float64(x)
	case int64:
		size = float64(x)
	default:
		if text := strings.TrimSpace(stringValue(value)); text != "" {
			return text + " байт"
		}
		return ""
	}
	units := []string{"байт", "KB", "MB", "GB"}
	idx := 0
	for size >= 1024 && idx < len(units)-1 {
		size /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%.0f %s", size, units[idx])
	}
	return fmt.Sprintf("%.1f %s", size, units[idx])
}

func statCardsFromMap(m map[string]any, fallback []statCard) []statCard {
	if len(m) == 0 {
		return fallback
	}
	cards := []statCard{}
	for _, key := range sortedKeys(m) {
		if len(cards) >= 4 {
			break
		}
		value := stringValue(m[key])
		if strings.TrimSpace(value) == "" {
			continue
		}
		cards = append(cards, statCard{Label: key, Value: value, Sub: "summary"})
	}
	if len(cards) == 0 {
		return fallback
	}
	return cards
}

func processStateText(status StatusResponse) string {
	if status.Running {
		return "Сервер запущен"
	}
	if status.ShutdownRequested {
		return "Остановка запрошена"
	}
	if status.ProbePending {
		return "Проверка состояния"
	}
	return "Сервер остановлен"
}
