package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// detailView shows a single object's YAML in a scrollable viewport with
// theme-aware syntax highlighting. It keeps the raw YAML so it can re-render or
// apply a yq filter.
type detailView struct {
	th    Theme
	vp    viewport.Model
	title string
	raw   string // original YAML, for re-highlight and yq filtering
	ready bool
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
	d.ready = true
}

// setYAML stores and renders highlighted YAML.
func (d *detailView) setYAML(title, yaml string) {
	d.title = title
	d.raw = yaml
	d.vp.SetContent(highlightYAML(yaml, d.th))
	d.vp.GotoTop()
	d.ready = true
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
	pct := d.th.Dim.Render(scrollPercent(d.vp.ScrollPercent()))
	title := d.th.ModalTitle.Render(d.title)
	gap := d.vp.Width - lipgloss.Width(title) - lipgloss.Width(pct)
	if gap < 1 {
		gap = 1
	}
	header := title + strings.Repeat(" ", gap) + pct
	return header + "\n" + d.vp.View()
}

func scrollPercent(f float64) string {
	p := int(f * 100)
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return itoa(p) + "%"
}
