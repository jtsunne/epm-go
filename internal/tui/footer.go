package tui

// renderFooter renders the key binding help footer at full terminal width.
// When app.showHelp is true, shows all key bindings; otherwise a brief hint.
func renderFooter(app *App) string {
	width := app.width
	if width <= 0 {
		width = 80
	}
	text := "? for help"
	if app.showHelp {
		text = helpText
	}
	return StyleDim.Width(width).Render(text)
}
