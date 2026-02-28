package tui

// renderFooter renders the key binding help footer at full terminal width.
// When app.deleteStatus is set, it is shown instead of the normal help text.
// When app.showHelp is true, shows all key bindings; otherwise a brief hint.
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
	text := "? for help"
	if app.showHelp {
		text = helpText
	}
	return StyleDim.Width(width).Render(text)
}
