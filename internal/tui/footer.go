package tui

// renderFooter renders the key binding help footer.
// Placeholder — full implementation in Task 8.
func renderFooter(app *App) string {
	if app.showHelp {
		return StyleDim.Render(helpText)
	}
	return StyleDim.Render("? for help")
}
