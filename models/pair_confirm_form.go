package models

import (
	"bluepala/common"
	"bluepala/config"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PairConfirmForm struct {
	Device       *common.Device
	Passkey      uint32
	ConfirmValue bool
	Colors       config.Colors
}

func ModelPairConfirmForm(colors config.Colors) PairConfirmForm {
	return PairConfirmForm{
		ConfirmValue: false,
		Colors:       colors,
	}
}

func (m PairConfirmForm) Init() tea.Cmd {
	return nil
}

func (m PairConfirmForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "tab", "shift+tab", "left", "right":
			m.ConfirmValue = !m.ConfirmValue
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return common.SubmitConfirmMsg{Confirmed: false} }
		case "enter":
			confirmed := m.ConfirmValue
			return m, func() tea.Msg { return common.SubmitConfirmMsg{Confirmed: confirmed} }
		}
	}

	return m, nil
}

func (m PairConfirmForm) View() string {
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Active)).
		Foreground(lipgloss.Color(m.Colors.Primary)).
		Align(lipgloss.Center).
		Padding(0, 1).
		Width(50)

	inactiveBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.Inactive)).
		Align(lipgloss.Center).
		Padding(0, 3).
		Width(18)

	activeBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.Colors.ActiveText)).
		Align(lipgloss.Center).
		Padding(0, 3).
		Width(18)

	confirmButton := inactiveBorderStyle.Render("Confirm")
	cancelButton := activeBorderStyle.Render("Cancel")

	if m.ConfirmValue {
		confirmButton = activeBorderStyle.Render("Confirm")
		cancelButton = inactiveBorderStyle.Render("Cancel")
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.Colors.ActiveText)).
		Width(44).
		Align(lipgloss.Center)

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Inactive)).
		Width(44)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Active)).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.Colors.Primary))

	passkeyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.Colors.ActiveText)).
		Width(44).
		Align(lipgloss.Center)

	deviceName := ""
	deviceAddr := ""
	deviceType := ""
	if m.Device != nil {
		deviceName = m.Device.Name
		if lipgloss.Width(deviceName) > 40 {
			deviceName = deviceName[:37] + "..."
		}
		deviceAddr = m.Device.Address
		deviceType = m.Device.Icon
	}

	addrRow := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Width(10).Render("MAC"),
		valueStyle.Render(deviceAddr),
	)

	typeRow := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Width(10).Render("Type"),
		valueStyle.Width(34).Render(deviceType),
	)

	return containerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(deviceName),
			dividerStyle.Render(strings.Repeat("─", 44)),
			addrRow,
			typeRow,
			"",
			passkeyStyle.Render(fmt.Sprintf("Passkey: %06d", m.Passkey)),
			"",
			lipgloss.JoinHorizontal(lipgloss.Center,
				cancelButton, confirmButton,
			),
		),
	)
}
