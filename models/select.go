// models/select.go
package models

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type Select struct {
	Options  []string
	Selected int
}

func (m *Select) Init() tea.Cmd { 
	m.Selected = 0
	return nil
}

func (m *Select) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Selected > 0 {
				m.Selected--
			}
		case "down", "j":
			if m.Selected < len(m.Options)-1 {
				m.Selected++
			}
		}
	}
	return nil
}

func (m *Select) View() string {
	if len(m.Options) == 0 {
		return "(no adapters found)"
	}

	out := ""
	for i, opt := range m.Options {
		cursor := "    "
		if i == m.Selected {
			cursor = "  > "
		}
		out += fmt.Sprintf("%s%s\n", cursor, opt)
	}
	return out
}
