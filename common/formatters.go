package common

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/godbus/dbus/v5"
)

func CalculatePadding(s string) int {
	totalWidth := WindowDimensions().Width
	line := strings.Split(s, "\n")[0]

	// Use lipgloss.Width to correctly calculate visible width, ignoring ANSI codes
	textWidth := lipgloss.Width(line)

	// Calculate padding and ensure it's not negative
	return max(0, (totalWidth-textWidth)/2)
}

func JustifyBetween(s1, s2 string, padding int) string {
	totalWidth := WindowDimensions().Width
	spaceWidth := max(totalWidth - lipgloss.Width(s1) - lipgloss.Width(s2), 0)

	return s1 + strings.Repeat(" ", spaceWidth-padding) + s2
}

func CalcTitle(title string, selected bool, totalWidth int) string {
	color := "#a7abca"
	bold := false
	if selected {
		color = "#9cca69"
		bold = true
	}

	if (totalWidth == 0) {
		totalWidth = WindowDimensions().Width
	}

	repeatCount := max(totalWidth-4-len(title), 0)
	return lipgloss.NewStyle().
		Bold(bold).
		Foreground(lipgloss.Color(color)).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("┌ %s %s┐", title, strings.Repeat("─", repeatCount)))
}

func FormatAdapters(conn *dbus.Conn, adapters []Adapter, width int) [][]string {
	data := [][]string{
		padHeaders([]string{"Name", "Address", "Powered", "Discoverable"}, []int{-1, -1, -1, -1}, width, nil), {""},
	}
	for _, d := range adapters {
		powered := "Off"
		if d.Powered {
			powered = "On"
		}

		discoverable := "No"
		if d.Discoverable {
			discoverable = "Yes"
		}

		row := []string{strings.ReplaceAll(string(d.Path), "/org/bluez/", ""), d.Address, powered, discoverable}
		data = append(data, row)
	}
	return data
}

func FormatDevices(devices []Device, selectedRow int, width int, height int) [][]string {
	align := lipgloss.Left
	data := [][]string{
		padHeaders([]string{"", "Type", "Name", "Connected"}, []int{5, 15, -1, 11}, width, &align), {""},
	}

	for i, d := range devices {
		connected := "    ○    "
		if d.Connected {
			connected = "    ●    "
		}

		devType := fmt.Sprintf("[%s]", d.Icon)

		row := []string{"   ", devType, d.Name, connected}
		if (i == min(selectedRow, height-2)) {
			row = []string{" > ", devType, d.Name, connected}
		}

		for j, str := range row {
			if lipgloss.Width(str) > lipgloss.Width(data[0][j]) {
				row[j] = str[:max(0, lipgloss.Width(data[0][j])-3)] + "..."
			}
		}

		data = append(data, row)
	}

	for range (height - len(data)) {
		data = append(data, []string{""})
	}
	
	return data
}

func FormatDetails(device *Device, width int, height int, selectedBox bool, selectedRow int) [][]string {
	data := [][]string{}

	connected := "Disconnected"
	if (device.Connected) {
		connected = "Connected"
	}

	alignVal := lipgloss.Center
	style := lipgloss.NewStyle().Bold(true)
	if selectedBox {
		style = style.Foreground(lipgloss.Color("#cda162"))
	}

	options := []string{"[Toggle]", "[Remove]", "[Rename]"}
	options[selectedRow] = style.Render(options[selectedRow])

	spacingWidth := width - lipgloss.Width(strings.Join(options, "")) - 4
	spacing := []string{
		strings.Repeat(" ", spacingWidth - (spacingWidth / 2)),
		strings.Repeat(" ", spacingWidth / 2),
	}

	data = append(data, []string{""})
	data = append(data, padHeaders([]string{style.Render(device.Name)}, []int{-1}, width, &alignVal))
	data = append(data, []string{""})
	data = append(data, []string{"MAC: " + device.Address})
	data = append(data, []string{"Status: " + connected})
	data = append(data, []string{"Battery: " + fmt.Sprintf("%d%%", device.Battery)})
	data = append(data, []string{"RSSI: " + fmt.Sprintf("%d%%", device.RSSI)})
	data = append(data, []string{"Type: " + device.Icon})
	data = append(data, []string{strings.Repeat("-", width - 4)})
	data = append(data, []string{fmt.Sprintf("%s%s%s%s%s", options[0], spacing[0], options[1], spacing[1], options[2])})
	data = append(data, []string{""})

	return data
}

func FormatArrays(arr []Device, selectedIndex int, windowSize int) []Device {
	start := 0
	if selectedIndex >= windowSize {
		start = selectedIndex - windowSize + 1
	}
	end := start + windowSize
	if end > len(arr) {
		end = len(arr)
		start = max(end-windowSize, 0)
	}
	if start > end {
		start = end
	}
	return arr[start:end]
}

func SanitizeEmojis(s, replacement string) string {
	// Unicode regex range for emojis — covers most common sets (Emoticons, Misc Symbols, Transport, etc.)
	re := regexp.MustCompile(`[\p{So}\p{Sk}\p{Cs}\x{1F000}-\x{1FAFF}\x{2600}-\x{27BF}\x{1F300}-\x{1F6FF}]+`)
	return re.ReplaceAllString(s, replacement)
}