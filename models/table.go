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
	case common.AdapterUpdateMsg:
		m.Adapters = msg
	case common.DeviceUpdateMsg:
		paired := make([]common.Device, 0)
		scanned := make([]common.Device, 0)

		for _, d := range msg {
			if (d.Paired) {
				paired = append(paired, d)
			} else if (d.Connectable) {
				scanned = append(scanned, d)
			}
		}

		if (m.PairedDevices != nil) {
			m.PairedDevices = paired
		} else if (m.SelectedPaired != nil) {
			m.ScannedDevices = scanned
		}

	case common.DeviceSelectedMsg:
		m.SelectedPaired = &msg.Device

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if !m.IsTableSelected {
				return m, nil
			}

			if m.SelectedRow > 0 {
				m.SelectedRow--

				if (m.PairedDevices != nil && m.SelectedRow < len(m.PairedDevices)) {
					cmd := func() tea.Cmd {
						return func() tea.Msg {
							return common.DeviceSelectedMsg{Device: m.PairedDevices[m.SelectedRow]}
						}
					}()

					return m, cmd
				}
			} 
		case "down", "j":
			if !m.IsTableSelected {
				return m, nil
			}

			var maxRows int
			if m.PairedDevices != nil {
				maxRows = len(m.PairedDevices) - 1
			} else if m.Adapters != nil {
				maxRows = len(m.Adapters) - 1
			}

			m.SelectedRow = (m.SelectedRow + 1 ) % (maxRows + 1)

			if (m.PairedDevices != nil && m.SelectedRow < len(m.PairedDevices)) {
				cmd := func() tea.Cmd {
					return func() tea.Msg {
						return common.DeviceSelectedMsg{Device: m.PairedDevices[m.SelectedRow]}
					}
				}()

				return m, cmd
			}

		case "left", "h":
			if (m.SelectedPaired != nil) {
				m.SelectedRow--
				if (m.SelectedRow < 0) {
					m.SelectedRow = 2
				}
			}
		case "right", "l":
			if (m.SelectedPaired != nil) {
				m.SelectedRow++
				if (m.SelectedRow > 2) {
					m.SelectedRow = 0
				}
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
