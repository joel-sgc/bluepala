package main

import (
	"bluepala/common"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type backgroundModel struct {
	data *BluepalaData
}

func (b backgroundModel) Init() tea.Cmd {
	return nil
}

func (b backgroundModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return b, nil
}

func (b backgroundModel) View() string {
	return b.data.MainView() // Only render main view, no modal logic
}

func (m *BluepalaData) MainView() string {
	var rightPane string
	devicesWidth := m.Width

	if m.DetailsTable.SelectedPaired != nil {
		rightPane = m.DetailsTable.View()
		devicesWidth = m.Width - 36
	}

	m.DevicesTable.Width = devicesWidth

	return lipgloss.JoinVertical(lipgloss.Left,
		strings.TrimSpace(common.HJoin(
			m.DevicesTable.View(),
			rightPane,
			m.DevicesTable.Width,
			m.DetailsTable.Width,
		)),
		m.ScannedTable.View(),
		m.AdapterTable.View(),
		m.StatusBar.View(),
	)
}
