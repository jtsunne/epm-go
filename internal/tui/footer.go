package tui

// renderFooter renders the key binding help footer at full terminal width.
// When app.deleteStatus or app.settingsStatus is set, it is shown instead of
// the normal help text. When app.showHelp is true, shows all key bindings.
func renderFooter(app *App) string {
	width := app.width
	if width <= 0 {
		width = 80
	}
	if app.deleteStatus != "" {
		if app.deleteStatusErr {
			return StyleError.Width(width).Render(app.deleteStatus)
		}
		return StyleGreen.Width(width).Render(app.deleteStatus)
	}
	if app.settingsStatus != "" {
		if app.settingsStatusErr {
			return StyleError.Width(width).Render(app.settingsStatus)
		}
		return StyleGreen.Width(width).Render(app.settingsStatus)
	}
	text := "? for help"
	if app.showHelp {
		text = helpText
	}
	return StyleDim.Width(width).Render(text)
}
