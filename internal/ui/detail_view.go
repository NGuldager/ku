package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// detailView shows a single object's YAML in a scrollable viewport with
// theme-aware syntax highlighting. It keeps the raw YAML so it can re-render or
// apply a yq filter.
type detailView struct {
	th    Theme
	vp    viewport.Model
	title string
	raw   string // original YAML, for re-highlight and yq filtering
}

func newDetailView(th Theme) detailView {
	return detailView{th: th, vp: viewport.New(0, 0)}
}

func (d *detailView) setSize(w, h int) {
	if h < 1 {
		h = 1
	}
	d.vp.Width = w
	d.vp.Height = h - 1 // leave a row for the title
}

// setMessage shows plain (unhighlighted) text such as "loading…" or an error.
func (d *detailView) setMessage(title, body string) {
	d.title = title
	d.raw = ""
	d.vp.SetContent(body)
	d.vp.GotoTop()
}

// setYAML stores and renders highlighted YAML.
func (d *detailView) setYAML(title, yaml string) {
	d.title = title
	d.raw = yaml
	d.vp.SetContent(highlightYAML(yaml, d.th))
	d.vp.GotoTop()
}

// setFiltered renders a yq-filtered result without discarding the raw YAML.
func (d *detailView) setFiltered(yaml string) {
	d.vp.SetContent(highlightYAML(yaml, d.th))
	d.vp.GotoTop()
}

func (d detailView) Update(msg tea.Msg) (detailView, tea.Cmd) {
	var cmd tea.Cmd
	d.vp, cmd = d.vp.Update(msg)
	return d, cmd
}

func (d detailView) View() string {
	title := d.th.ModalTitle.Render(d.title)
	pct := d.th.Dim.Render(scrollPercent(d.vp.ScrollPercent()))
	return spread(title, pct, d.vp.Width) + "\n" + d.vp.View()
}

func scrollPercent(f float64) string {
	return itoa(clamp(int(f*100), 0, 100)) + "%"
}
