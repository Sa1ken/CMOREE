//go:build nativeui

package main

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
)

func (a *DesktopApp) renderPageContent(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	if p.Version != p.Applied {
		p.Body.SetText(p.Text)
		p.Applied = p.Version
	}
	switch def.ID {
	case "dashboard":
		return a.dashboardPage(gtx, p)
	case "console":
		return a.consolePage(gtx, def, p)
	case "kits":
		return a.kitsPage(gtx, def, p)
	case "configs":
		return a.configsPage(gtx, def, p)
	case "files":
		return a.filesPage(gtx, def, p)
	case "addons":
		return a.addonsPage(gtx, def, p)
	case "logs":
		return a.logsPage(gtx, def, p)
	case "database":
		return a.databasePage(gtx, def, p)
	case "backups":
		return a.backupsPage(gtx, def, p)
	case "discord":
		return a.discordPage(gtx, def, p)
	case "scheduler":
		return a.schedulerPage(gtx, def, p)
	case "accounts":
		return a.accountsPage(gtx, def, p)
	case "audit":
		return a.auditPage(gtx, def, p)
	default:
		return a.genericPage(gtx, def, p)
	}
}

func (a *DesktopApp) dashboardPage(gtx layout.Context, p *PageRuntime) layout.Dimensions {
	status := a.status
	events := slicePath(p.Data, "audit", "items")
	history := slicePath(p.Data, "console_history", "items")
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dashboardHero(gtx, p, status)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dashboardQuickActions(gtx, p)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(.38, func(gtx layout.Context) layout.Dimensions {
					return a.cardsPanel(gtx, p, "Игроки сейчас", fmt.Sprintf("%s / %s", stringValue(status.Server.PlayersOnline), stringValue(status.Server.PlayersMax)), cardsFromItems("player", status.Server.PlayersList, ""), "Сервер остановлен. После запуска здесь появится живой список игроков.")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 14, 1) }),
				layout.Flexed(.62, func(gtx layout.Context) layout.Dimensions {
					if len(events) == 0 {
						return a.consoleLogPanel(gtx, p, history)
					}
					return a.cardsPanel(gtx, p, "Последние события", countLabel(events), cardsFromItems("audit", events, ""), "Событий пока нет.")
				}),
			)
		}),
	)
}

type statCard struct {
	Label string
	Value string
	Sub   string
	OK    bool
}

func (a *DesktopApp) statGrid(gtx layout.Context, cards []statCard) layout.Dimensions {
	cols := 7
	if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(920)) {
		cols = 2
	} else if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(1280)) {
		cols = 4
	}
	rows := (len(cards) + cols - 1) / cols
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, func() []layout.FlexChild {
		out := []layout.FlexChild{}
		for r := 0; r < rows; r++ {
			row := r
			out = append(out, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{}
				for c := 0; c < cols; c++ {
					i := row*cols + c
					if i >= len(cards) {
						children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }))
						continue
					}
					card := cards[i]
					children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Right: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.statCard(gtx, card)
						})
					}))
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
			}))
		}
		return out
	}()...)
}

func (a *DesktopApp) statCard(gtx layout.Context, card statCard) layout.Dimensions {
	return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(112)), func(gtx layout.Context) layout.Dimensions {
		return panel(gtx, func(gtx layout.Context) layout.Dimensions {
			valueColor := palette.text
			if card.OK {
				valueColor = palette.success
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, strings.ToUpper(card.Label), 10, palette.muted, true)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, emptyDash(card.Value), 18, valueColor, true)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, emptyDash(card.Sub), 10, palette.dim, false)
				}),
			)
		})
	})
}

func (a *DesktopApp) genericPage(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	if pageNeedsEditor(def.ID) {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
			layout.Flexed(.45, func(gtx layout.Context) layout.Dimensions { return a.summaryPanel(gtx, def, p) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
			layout.Flexed(.55, func(gtx layout.Context) layout.Dimensions { return a.editorPanel(gtx, def, p) }),
		)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.pageControls(gtx, def, p) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 14) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.summaryPanel(gtx, def, p)
		}),
	)
}

func (a *DesktopApp) pageControls(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(.33, func(gtx layout.Context) layout.Dimensions {
						return a.editorBox(gtx, &p.Search, searchHint(def.ID), true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
					layout.Flexed(.34, func(gtx layout.Context) layout.Dimensions {
						return a.editorBox(gtx, &p.Primary, primaryHint(def.ID), true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
					layout.Flexed(.33, func(gtx layout.Context) layout.Dimensions {
						return a.editorBox(gtx, &p.Secondary, secondaryHint(def.ID), true)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 10) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, a.actionButtonsFor(def.ID, p)...)
			}),
		)
	})
}

func (a *DesktopApp) actionButtonsFor(id string, p *PageRuntime) []layout.FlexChild {
	spec := map[string][][3]string{
		"console":   {{"refresh", "Обновить", "ghost"}, {"send", "Отправить", "primary"}, {"search", "Каталог", "ghost"}},
		"kits":      {{"refresh", "Обновить", "ghost"}, {"search", "Каталог", "ghost"}, {"issue", "Выдать", "success"}, {"save", "Сохранить JSON", "primary"}, {"delete", "Удалить", "danger"}},
		"configs":   {{"refresh", "Обновить", "ghost"}, {"read", "Прочитать raw", "primary"}, {"write", "Записать raw", "warning"}, {"save", "Сохранить panel_config", "primary"}},
		"files":     {{"refresh", "Список", "ghost"}, {"read", "Прочитать", "primary"}, {"write", "Записать", "warning"}, {"upload", "Загрузить", "primary"}, {"save", "Папка", "ghost"}, {"delete", "Удалить", "danger"}},
		"addons":    {{"refresh", "Обновить", "ghost"}, {"search", "Поиск Steam", "primary"}, {"read", "ModInfo", "ghost"}, {"run", "Скачать IDs", "success"}, {"save", "Сохранить Mods JSON", "warning"}},
		"logs":      {{"refresh", "Обновить", "ghost"}, {"read", "Читать путь", "primary"}},
		"database":  {{"refresh", "Обновить", "ghost"}, {"read", "Таблицы/строки", "primary"}, {"snapshot", "Snapshot", "success"}, {"diff", "Diff", "warning"}},
		"backups":   {{"refresh", "Обновить", "ghost"}, {"upload", "Загрузить ZIP", "primary"}, {"restore", "Откатить", "warning"}, {"wipe", "Вайп", "danger"}},
		"discord":   {{"refresh", "Обновить", "ghost"}, {"test", "Тест", "primary"}, {"save", "Сохранить JSON", "warning"}, {"run", "Run task", "success"}, {"delete", "Delete task", "danger"}},
		"scheduler": {{"refresh", "Обновить", "ghost"}, {"save", "Сохранить JSON", "primary"}, {"toggle", "Toggle", "warning"}, {"run", "Run", "success"}, {"delete", "Delete", "danger"}},
		"accounts":  {{"refresh", "Обновить", "ghost"}, {"save", "Сохранить JSON", "primary"}, {"read", "Пароль", "ghost"}, {"write", "Сменить пароль", "warning"}, {"toggle", "Toggle", "warning"}, {"delete", "Delete", "danger"}},
		"audit":     {{"refresh", "Обновить", "ghost"}},
	}[id]
	children := []layout.FlexChild{}
	for _, item := range spec {
		name, label, tone := item[0], item[1], item[2]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.actionButton(gtx, p, name, label, tone)
		}))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }))
	}
	children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.editorBox(gtx, &p.Action, "DELETE / WIPE", true) }))
	return children
}

func (a *DesktopApp) actionButton(gtx layout.Context, p *PageRuntime, name, label, tone string) layout.Dimensions {
	btn := p.Buttons[name]
	if btn == nil {
		btn = new(widget.Clickable)
		p.Buttons[name] = btn
	}
	if btn.Clicked(gtx) {
		a.pageAction(p.ID, name)
	}
	return a.button(gtx, btn, label, tone)
}

func (a *DesktopApp) summaryPanel(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "Сводка", 15, palette.text, true)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 10) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				items := extractList(p.Data)
				if len(items) == 0 {
					return a.dataPanel(gtx, "Данные", p.Data)
				}
				return a.itemGrid(gtx, p, items)
			}),
		)
	})
}

func (a *DesktopApp) itemGrid(gtx layout.Context, p *PageRuntime, items []any) layout.Dimensions {
	cols := 4
	if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(820)) {
		cols = 1
	} else if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(1180)) {
		cols = 2
	} else if gtx.Constraints.Max.X < gtx.Dp(unit.Dp(1520)) {
		cols = 3
	}
	rows := (len(items) + cols - 1) / cols
	return p.List.Layout(gtx, rows, func(gtx layout.Context, row int) layout.Dimensions {
		children := []layout.FlexChild{}
		for col := 0; col < cols; col++ {
			index := row*cols + col
			if index >= len(items) {
				children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }))
				continue
			}
			item := items[index]
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Right: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.itemCard(gtx, item)
				})
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

func (a *DesktopApp) itemCard(gtx layout.Context, item any) layout.Dimensions {
	return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(72)), func(gtx layout.Context) layout.Dimensions {
		return rounded(gtx, palette.surface2, 12, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, titleFor(item), 13, palette.text, true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, subtitleFor(item), 11, palette.muted, false)
					}),
				)
			})
		})
	})
}

func (a *DesktopApp) editorPanel(gtx layout.Context, def PageDef, p *PageRuntime) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "JSON / редактор", 15, palette.text, true)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, editorHint(def.ID), 11, palette.muted, false)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 10) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.editorBox(gtx, &p.Body, "JSON или текст", false)
			}),
		)
	})
}

func (a *DesktopApp) dataPanel(gtx layout.Context, title string, data map[string]any) layout.Dimensions {
	return panel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, title, 15, palette.text, true) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 10) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				text := pretty(data)
				lbl := a.label(gtx, text, 11, palette.muted, false)
				return lbl
			}),
		)
	})
}

func extractList(data map[string]any) []any {
	for _, key := range []string{"items", "users", "files", "groups", "roots", "tables", "players", "lines"} {
		if items := sliceValue(data[key]); len(items) > 0 {
			return items
		}
	}
	for _, key := range sortedKeys(data) {
		if nested := mapValue(data[key]); nested != nil {
			if items := extractList(nested); len(items) > 0 {
				return items
			}
		}
	}
	return nil
}

func emptyDash(v string) string {
	if strings.TrimSpace(v) == "" || v == "<nil>" {
		return "-"
	}
	return v
}

func searchHint(id string) string {
	switch id {
	case "console":
		return "поиск команд/предметов"
	case "kits":
		return "поиск предметов"
	case "addons":
		return "Steam search / Workshop ID"
	case "database":
		return "поиск строк"
	case "backups":
		return "поиск бэкапов"
	default:
		return "фильтр"
	}
}

func primaryHint(id string) string {
	switch id {
	case "console":
		return "команда"
	case "kits":
		return "kit id"
	case "files", "logs", "database", "configs":
		return "путь"
	case "addons":
		return "Workshop IDs"
	case "discord":
		return "task id"
	case "scheduler":
		return "schedule id"
	case "accounts":
		return "username"
	case "backups":
		return "backup.zip или режим"
	default:
		return "основное значение"
	}
}

func secondaryHint(id string) string {
	switch id {
	case "kits":
		return "игрок"
	case "database":
		return "таблица"
	case "files":
		return "имя папки"
	case "accounts":
		return "новый пароль"
	case "backups":
		return "db,Saves,Logs"
	default:
		return "доп. значение"
	}
}

func editorHint(id string) string {
	switch id {
	case "console":
		return "Здесь отображается история и каталог. Команду вводите в основное поле."
	case "configs", "files":
		return "После чтения файла здесь будет его содержимое; кнопка записи отправит текст обратно."
	case "backups":
		return "Для отката выберите бэкап и введите RESTORE. Для вайпа введите WIPE; цели можно указать через запятую."
	default:
		return "Можно редактировать JSON для save-действий. Для удаления используйте подтверждение DELETE."
	}
}

func pageNeedsEditor(id string) bool {
	switch id {
	case "configs", "files":
		return true
	default:
		return false
	}
}
