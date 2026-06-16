package ui

import (
	tea "charm.land/bubbletea/v2"
)

// Options configures a kli session.
type Options struct {
	Context    string
	Namespace  string
	Resource   string
	Theme      string
	Kubeconfig string // explicit kubeconfig path ("" = default lookup)
}

// Run starts the interactive TUI. The cluster connection and config load run in
// the background behind a splash screen (see startupCmd / adoptStartup); flags
// take precedence over the remembered context/namespace from the last session.
func Run(opts Options) error {
	th := PickTheme(opts.Theme)
	saved, hasSaved := loadState()

	app := App{theme: th, keys: defaultKeys(), splash: true, opts: opts, saved: saved, hasSaved: hasSaved}
	app.spin = newSpinner(th)

	m, err := tea.NewProgram(app).Run()
	if err != nil {
		return err
	}
	// A fatal connection error is reported here rather than from a goroutine.
	if fin, ok := m.(App); ok && fin.startErr != nil {
		return fin.startErr
	}
	return nil
}
