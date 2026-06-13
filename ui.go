//go:build nativeui

package main

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var palette = struct {
	bg, bgSoft, surface, surface2, border, border2     color.NRGBA
	text, muted, dim, accent, success, danger, warning color.NRGBA
}{
	bg:       rgb(13, 20, 29),
	bgSoft:   rgb(17, 27, 39),
	surface:  rgb(19, 33, 48),
	surface2: rgb(24, 41, 59),
	border:   rgba(157, 180, 209, 42),
	border2:  rgba(157, 180, 209, 72),
	text:     rgb(239, 245, 251),
	muted:    rgb(143, 163, 188),
	dim:      rgb(103, 122, 145),
	accent:   rgb(115, 174, 255),
	success:  rgb(104, 196, 154),
	danger:   rgb(229, 132, 140),
	warning:  rgb(239, 184, 108),
}

func rgb(r, g, b uint8) color.NRGBA     { return color.NRGBA{R: r, G: g, B: b, A: 255} }
func rgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

func (a *DesktopApp) layout(gtx layout.Context) layout.Dimensions {
	fill(gtx, palette.bg)
	switch a.mode {
	case "app":
		return a.layoutApp(gtx)
	case "setup":
		return a.layoutSetup(gtx)
	case "login":
		return a.layoutLogin(gtx)
	default:
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, "Подключение к CMoree...", 18, palette.text, true)
		})
	}
}

func (a *DesktopApp) layoutApp(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.topbar(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return force(gtx, gtx.Dp(unit.Dp(234)), gtx.Constraints.Max.Y, a.sidebar)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.page(gtx)
				}),
			)
		}),
	)
}

func (a *DesktopApp) topbar(gtx layout.Context) layout.Dimensions {
	return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(62)), func(gtx layout.Context) layout.Dimensions {
		fill(gtx, rgb(15, 24, 35))
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.logo != nil {
						return force(gtx, gtx.Dp(unit.Dp(30)), gtx.Dp(unit.Dp(30)), a.logo.Layout)
					}
					return a.badge(gtx, "C", palette.accent)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 8, 1) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, "Moree", 18, palette.text, true) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 18, 1) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.label(gtx, stringValue(a.bootstrap.Config["server_name"]), 12, palette.muted, false)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					status := "Offline"
					tone := palette.danger
					if a.status.Running {
						status = "Online"
						tone = palette.success
					}
					return a.pill(gtx, status, tone)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 12, 1) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					name := a.bootstrap.User.DisplayName
					if name == "" {
						name = a.bootstrap.User.Username
					}
					return a.userChip(gtx, name, a.bootstrap.User.RankLabel)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 10, 1) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.button(gtx, &a.logoutButton, "Выйти", "ghost")
				}),
			)
		})
	})
}

func (a *DesktopApp) sidebar(gtx layout.Context) layout.Dimensions {
	fill(gtx, rgb(16, 27, 40))
	return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.editorBox(gtx, &a.sidebarSearch, "Фильтр разделов", true)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 12) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				groups := groupedPages()
				return a.sidebarList.Layout(gtx, len(groups), func(gtx layout.Context, index int) layout.Dimensions {
					group := groups[index]
					return a.sidebarGroup(gtx, group)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, "CMoree native client", 10, palette.dim, false)
			}),
		)
	})
}

func (a *DesktopApp) sidebarGroup(gtx layout.Context, group string) layout.Dimensions {
	query := stringsLower(a.sidebarSearch.Text())
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(4), Left: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, group, 10, palette.muted, true)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{}
			for _, page := range pageDefs {
				if page.Group != group || !a.pageAllowed(page) {
					continue
				}
				if query != "" && !stringsContains(stringsLower(page.Label+" "+page.Subtitle), query) {
					continue
				}
				p := page
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.navItem(gtx, p)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

func (a *DesktopApp) pageAllowed(page PageDef) bool {
	if a.bootstrap.IsRoot {
		return true
	}
	if allowed, ok := a.bootstrap.Nav[page.ID]; ok {
		return allowed
	}
	return hasAny(page.Perms, permissionSet(a.bootstrap.Permissions), false)
}

func (a *DesktopApp) navItem(gtx layout.Context, page PageDef) layout.Dimensions {
	rt := a.pages[page.ID]
	active := a.activePage == page.ID
	btn := rt.Buttons["nav"]
	if btn == nil {
		btn = new(widget.Clickable)
		rt.Buttons["nav"] = btn
	}
	if btn.Clicked(gtx) {
		a.activePage = page.ID
		a.loadPage(page.ID)
	}
	bg := color.NRGBA{}
	txt := palette.muted
	if active {
		bg = rgba(115, 174, 255, 36)
		txt = palette.text
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(40)), func(gtx layout.Context) layout.Dimensions {
			return rounded(gtx, bg, 13, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.icon(gtx, page.Icon, txt, 18) }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 10, 1) }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, page.Label, 13, txt, true) }),
					)
				})
			})
		})
	})
}

func (a *DesktopApp) page(gtx layout.Context) layout.Dimensions {
	def := pageByID(a.activePage)
	rt := a.pages[a.activePage]
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.pageHeader(gtx, def)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.contentList.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.renderPageContent(gtx, def, rt)
				})
			})
		}),
	)
}

func (a *DesktopApp) pageHeader(gtx layout.Context, def PageDef) layout.Dimensions {
	return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(78)), func(gtx layout.Context) layout.Dimensions {
		fill(gtx, rgb(15, 24, 35))
		return layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(12), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, def.Label, 20, palette.text, true) }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.label(gtx, def.Subtitle, 11, palette.muted, false)
						}),
					)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 1)}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.button(gtx, &a.refreshAll, "Обновить F5", "ghost")
				}),
			)
		})
	})
}

func (a *DesktopApp) button(gtx layout.Context, btn *widget.Clickable, textValue, tone string) layout.Dimensions {
	bg := rgba(255, 255, 255, 10)
	fg := palette.text
	switch tone {
	case "primary":
		bg = rgba(115, 174, 255, 48)
		fg = palette.text
	case "success":
		bg = rgba(104, 196, 154, 32)
		fg = palette.success
	case "danger":
		bg = rgba(229, 132, 140, 30)
		fg = palette.danger
	case "warning":
		bg = rgba(239, 184, 108, 30)
		fg = palette.warning
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return rounded(gtx, bg, 12, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, textValue, 13, fg, true)
			})
		})
	})
}

func (a *DesktopApp) label(gtx layout.Context, textValue string, size float32, col color.NRGBA, bold bool) layout.Dimensions {
	l := material.Label(a.theme, unit.Sp(size), textValue)
	l.Color = col
	l.MaxLines = 2
	l.Alignment = text.Start
	if bold {
		l.Font.Weight = 700
	}
	return l.Layout(gtx)
}

func (a *DesktopApp) editorBox(gtx layout.Context, ed *widget.Editor, hint string, single bool) layout.Dimensions {
	ed.SingleLine = single
	return rounded(gtx, rgba(255, 255, 255, 8), 12, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(11), Right: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			style := material.Editor(a.theme, ed, hint)
			style.Color = palette.text
			style.HintColor = palette.dim
			style.TextSize = unit.Sp(13)
			return style.Layout(gtx)
		})
	})
}

func (a *DesktopApp) icon(gtx layout.Context, data []byte, col color.NRGBA, size int) layout.Dimensions {
	ic, err := widget.NewIcon(data)
	if err != nil {
		return layout.Dimensions{}
	}
	return force(gtx, gtx.Dp(unit.Dp(size)), gtx.Dp(unit.Dp(size)), func(gtx layout.Context) layout.Dimensions {
		return ic.Layout(gtx, col)
	})
}

func fill(gtx layout.Context, col color.NRGBA) {
	paint.FillShape(gtx.Ops, col, clip.Rect{Max: gtx.Constraints.Max}.Op())
}

func rounded(gtx layout.Context, col color.NRGBA, radius int, child layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := child(gtx)
	call := macro.Stop()
	if col.A != 0 && dims.Size.X > 0 && dims.Size.Y > 0 {
		paint.FillShape(gtx.Ops, col, clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(radius))).Op(gtx.Ops))
	}
	call.Add(gtx.Ops)
	return dims
}

func panel(gtx layout.Context, child layout.Widget) layout.Dimensions {
	return rounded(gtx, palette.surface, 16, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, child)
	})
}

func spacer(gtx layout.Context, w, h int) layout.Dimensions {
	return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(w)), gtx.Dp(unit.Dp(h)))}
}

func (a *DesktopApp) pill(gtx layout.Context, textValue string, col color.NRGBA) layout.Dimensions {
	return rounded(gtx, color.NRGBA{R: col.R, G: col.G, B: col.B, A: 28}, 20, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.label(gtx, textValue, 12, col, true)
		})
	})
}

func (a *DesktopApp) badge(gtx layout.Context, textValue string, col color.NRGBA) layout.Dimensions {
	return force(gtx, gtx.Dp(unit.Dp(30)), gtx.Dp(unit.Dp(30)), func(gtx layout.Context) layout.Dimensions {
		return rounded(gtx, rgba(col.R, col.G, col.B, 46), 8, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.label(gtx, textValue, 14, col, true)
			})
		})
	})
}

func (a *DesktopApp) userChip(gtx layout.Context, name, role string) layout.Dimensions {
	if role == "" {
		role = "Администратор"
	}
	return rounded(gtx, rgba(255, 255, 255, 9), 15, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, name, 12, palette.text, true) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, role, 10, palette.accent, true) }),
			)
		})
	})
}

func stringsLower(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func stringsContains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

var _ = op.Record
