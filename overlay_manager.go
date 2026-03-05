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
	// Dynamic height calculation for ScannedTable only.
	// DevicesTable and DetailsTable keep their fixed initialized heights.
	// Each table block = CalcTitle (1 line) + rows (Height) + bottom border (1 line).
	// AdapterTable rows are not height-clamped; compute from actual adapter count.
	totalHeight := m.Height
	if totalHeight == 0 {
		totalHeight = common.WindowDimensions().Height
	}
	adapterRows := max(len(m.Adapters), 1) + 2 // header row + blank row + adapter rows
	adapterTotal := adapterRows + 2             // + CalcTitle + bottom border
	// Overhead: DevicesTable block + ScannedTable chrome (CalcTitle + border) + StatusBar
	fixedOverhead := (m.DevicesTable.Height + 2) + adapterTotal + 1 + 2
	m.ScannedTable.Height = max(totalHeight-fixedOverhead, 3)

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
