package models

import (
	"bluepala/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	godbus "github.com/godbus/dbus/v5"
)

type TableData struct {
	Conn            *godbus.Conn
	Title           string
	IsTableSelected bool
	SelectedRow     int
	Height          int
	Width						int

	Adapters        []common.Adapter
	PairedDevices   []common.Device
	SelectedPaired  *common.Device
	ScannedDevices	[]common.Device
}

func (m *TableData) Init() tea.Cmd {	
	return nil
}

func (m *TableData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// This table no longer processes device lists. The main model handles it.
	case common.DeviceSelectedMsg:
		m.SelectedPaired = msg.Device

	case tea.KeyMsg:
		if !m.IsTableSelected {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.SelectedRow > 0 {
				m.SelectedRow--
			}
		case "down", "j":
			var maxRows int
			if m.PairedDevices != nil {
				maxRows = len(m.PairedDevices)
			} else if m.ScannedDevices != nil {
				maxRows = len(m.ScannedDevices)
			} else if m.Adapters != nil {
				maxRows = len(m.Adapters)
			}

			// Prevent selection from going out of bounds
			if maxRows > 0 && m.SelectedRow < maxRows-1 {
				m.SelectedRow++
			}
		case "left", "h":
			if m.SelectedPaired != nil { // Details Table
				m.SelectedRow = (m.SelectedRow - 1 + 3) % 3
			}
		case "right", "l":
			if m.SelectedPaired != nil { // Details Table
				m.SelectedRow = (m.SelectedRow + 1) % 3
			}
		}

		// After moving, if it's a device table, send a selection message
		var selectedDevice *common.Device
		if len(m.PairedDevices) > 0 && m.SelectedRow < len(m.PairedDevices) {
			selectedDevice = &m.PairedDevices[m.SelectedRow]
		} else if len(m.ScannedDevices) > 0 && m.SelectedRow < len(m.ScannedDevices) {
			selectedDevice = &m.ScannedDevices[m.SelectedRow]
		}

		if selectedDevice != nil {
			return m, func() tea.Msg {
				return common.DeviceSelectedMsg{Device: selectedDevice}
			}
		}
	}

	return m, nil
}

func (m TableData) View() string {
	borderStyle := common.InactiveBorderStyle
	if m.IsTableSelected {
		borderStyle = common.ActiveBorderStyle
	}

	var tableData [][]string
	align := lipgloss.Center

	if m.Adapters != nil {
		tableData = common.FormatAdapters(m.Conn, m.Adapters, m.Width)
	} else if m.PairedDevices != nil {
		tableData = common.FormatDevices(
			common.FormatArrays(m.PairedDevices, m.SelectedRow, m.Height-1),
			m.SelectedRow, m.Width, m.Height,
		)
		align = lipgloss.Left
	} else if m.SelectedPaired != nil {
		align = lipgloss.Left
		tableData = common.FormatDetails(m.SelectedPaired, m.Width, m.Height, m.IsTableSelected, m.SelectedRow)
	} else if m.ScannedDevices != nil {
		tableData = common.FormatDevices(
			common.FormatArrays(m.ScannedDevices, m.SelectedRow, m.Height-1),
			m.SelectedRow, m.Width, m.Height,
		)
		align = lipgloss.Left
	}

	SelectedRow := -10
	if m.IsTableSelected && m.SelectedPaired == nil {
		SelectedRow = m.SelectedRow
	}
	
	table := table.New().
		Border(common.BoxBorder).
		BorderColumn(false).
		BorderStyle(borderStyle).
		StyleFunc(common.BoxStyle(SelectedRow, m.IsTableSelected, &align)).
		Rows(tableData...)

	return (common.CalcTitle(m.Title, m.IsTableSelected, m.Width) + table.Render())
}
