//go:build nativeui

package main

import (
	"image"

	"gioui.org/layout"
	"gioui.org/unit"
)

func (a *DesktopApp) layoutLogin(gtx layout.Context) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return force(gtx, gtx.Dp(unit.Dp(520)), gtx.Dp(unit.Dp(460)), func(gtx layout.Context) layout.Dimensions {
			return panel(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if a.logo != nil {
							return force(gtx, gtx.Dp(unit.Dp(58)), gtx.Dp(unit.Dp(58)), a.logo.Layout)
						}
						return a.badge(gtx, "C", palette.accent)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 12) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, "Moree", 26, palette.text, true) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "Нативная панель управления без WebView", 12, palette.muted, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 22) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return force(gtx, gtx.Dp(unit.Dp(440)), gtx.Dp(unit.Dp(44)), func(gtx layout.Context) layout.Dimensions {
							return a.editorBox(gtx, &a.loginServer, defaultPanelURL, true)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 10) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return force(gtx, gtx.Dp(unit.Dp(440)), gtx.Dp(unit.Dp(44)), func(gtx layout.Context) layout.Dimensions { return a.editorBox(gtx, &a.loginUser, "root", true) })
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 10) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return force(gtx, gtx.Dp(unit.Dp(440)), gtx.Dp(unit.Dp(44)), func(gtx layout.Context) layout.Dimensions {
							return a.editorBox(gtx, &a.loginPass, "Пароль", true)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 16) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.button(gtx, &a.loginButton, "Войти", "primary")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 10, 1) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.button(gtx, &a.checkButton, "Проверить", "ghost")
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 10, 1) }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.button(gtx, &a.setupButton, "Настройка", "warning")
							}),
						)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: image.Pt(1, gtx.Constraints.Max.Y)}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.toastView(gtx) }),
				)
			})
		})
	})
}

func (a *DesktopApp) layoutSetup(gtx layout.Context) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := minInt(gtx.Dp(unit.Dp(760)), gtx.Constraints.Max.X-gtx.Dp(unit.Dp(24)))
		h := minInt(gtx.Dp(unit.Dp(720)), gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(24)))
		if w < gtx.Dp(unit.Dp(520)) {
			w = gtx.Constraints.Max.X
		}
		if h < gtx.Dp(unit.Dp(520)) {
			h = gtx.Constraints.Max.Y
		}
		return force(gtx, w, h, func(gtx layout.Context) layout.Dimensions {
			return panel(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "Первичная настройка CMoree", 22, palette.text, true)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.label(gtx, "Заполните путь к серверу. Остальные поля можно добавить позже в разделе конфигов.", 12, palette.muted, false)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 16) }),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.setupList.Layout(gtx, len(setupFieldDefs), func(gtx layout.Context, index int) layout.Dimensions {
							f := setupFieldDefs[index]
							return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.label(gtx, f.Label, 11, palette.muted, true) }),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return force(gtx, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(42)), func(gtx layout.Context) layout.Dimensions {
											return a.editorBox(gtx, a.setupFields[f.Key], f.Hint, true)
										})
									}),
								)
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return spacer(gtx, 1, 8) }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.button(gtx, &a.setupSave, "Сохранить настройку", "primary")
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return a.toastView(gtx) }),
				)
			})
		})
	})
}

func (a *DesktopApp) toastView(gtx layout.Context) layout.Dimensions {
	if a.toast == "" {
		return layout.Dimensions{}
	}
	tone := palette.accent
	if a.toastKind == "err" {
		tone = palette.danger
	} else if a.toastKind == "ok" {
		tone = palette.success
	}
	return a.pill(gtx, a.toast, tone)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
