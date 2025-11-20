package common

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/godbus/dbus/v5"
	"golang.org/x/term"
)

var BlankDevice = Device{
	Path:        "-1",
	Name:        "-",
	Address:     "00:00:00:00:00:00",
	Icon:        "",
	AddressType: "-",
	Paired:      true,
	Trusted:     false,
	Connected:   false,
	Battery:     -1,
	Connectable: false,
	RSSI:        -1,
}

func WindowDimensions() struct{ Width, Height int } {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return struct{ Width, Height int }{80, 80}
	}
	return struct{ Width, Height int }{width, height}
}

func padHeaders(headers []string, headerLengths []int, totalWidth int, align *lipgloss.Position) []string {
	if len(headers) == 0 {
		return headers
	}

	// Fallback: if no lengths provided, auto-fill with -1 (flex)
	if headerLengths == nil || len(headerLengths) != len(headers) {
		headerLengths = make([]int, len(headers))
		for i := range headerLengths {
			headerLengths[i] = -1
		}
	}

	availableWidth := max(totalWidth-10, 1)

	// Calculate fixed width and identify flexible columns
	fixedWidth := 0
	flexColumns := []int{} // indices of flexible columns
	for i, w := range headerLengths {
		if w == -1 {
			flexColumns = append(flexColumns, i)
		} else {
			fixedWidth += w
		}
	}

	remaining := max(availableWidth-fixedWidth, 0)

	// Calculate base width and remainder for flexible columns
	flexCount := len(flexColumns)
	baseWidth := 0
	remainder := 0

	if flexCount > 0 {
		baseWidth = remaining / flexCount
		remainder = remaining % flexCount
	}

	// Distribute widths to flexible columns, handling remainder
	flexWidths := make([]int, flexCount)
	for i := range flexWidths {
		flexWidths[i] = baseWidth
		if i < remainder {
			flexWidths[i]++
		}
	}

	// Assign the calculated widths back to headerLengths
	for i, flexIndex := range flexColumns {
		headerLengths[flexIndex] = flexWidths[i]
	}

	if align == nil {
		pos := lipgloss.Center
		align = &pos
	}

	// Render headers with their respective widths
	finalHeaders := make([]string, len(headers))
	for i, h := range headers {
		width := max(headerLengths[i], 1)
		finalHeaders[i] = lipgloss.NewStyle().
			Width(width).
			Align(*align).
			Render(h)
	}

	return finalHeaders
}

var BoxBorder = lipgloss.Border{
	Top:      "",
	TopLeft:  "",
	TopRight: "",

	MiddleLeft:   "",
	MiddleRight:  "",
	Middle:       "",
	MiddleTop:    "",
	MiddleBottom: "─",

	Left: "│", Right: "│",
	BottomLeft: "└", Bottom: "─", BottomRight: "┘",
}
var ActiveBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cca69"))
var InactiveBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca"))

func BoxStyle(selectedRow int, selectedBox bool, align *lipgloss.Position, height int) func(row, col int) lipgloss.Style {
	padding := 1
	if align == nil {
		center := lipgloss.Center
		align = &center

		padding = 0
	}

	return func(row int, col int) lipgloss.Style {
		switch {
		case row == 0:
			return lipgloss.NewStyle().
				Bold(true).
				Foreground(func() lipgloss.Color {
					if selectedBox {
						return lipgloss.Color("#cda162")
					}
					return lipgloss.Color("#a7abca")
				}()).
				Align(*align).Padding(0, padding)
		case row == min(selectedRow+2, height) && selectedBox:
			return lipgloss.NewStyle().
				Background(lipgloss.Color("#a7abca")).
				Foreground(lipgloss.Color("#444a66")).
				Align(*align).Padding(0, padding)
		default:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#a7abca")).Align(*align).Padding(0, padding)
		}
	}
}

func HJoin(left, right string, leftW, rightW int) string {
	// Ensure each block is constrained to the requested width (wraps/truncates if needed).
	left = lipgloss.NewStyle().Width(leftW).Render(left)
	right = lipgloss.NewStyle().Width(rightW).Render(right)

	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	n := max(len(rightLines), len(leftLines))

	var b strings.Builder
	for i := range n {
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		// Pad left to exactly leftW (use fmt with -*s)
		// Note: fmt padding counts runes; if you need exact terminal cell widths for CJK/wide runes,
		// consider github.com/mattn/go-runewidth to pad to terminal width.
		b.WriteString(fmt.Sprintf("%-*s%s\n", leftW, l, r))
	}
	return b.String()
}

// Returns two slices: paired devices and unpaired devices
func FilterDevicesByPaired(devices []Device) ([]Device, []Device) {
	pairedDevices := make([]Device, 0)
	unpairedDevices := make([]Device, 0)

	for _, device := range devices {
		if device.Paired {
			pairedDevices = append(pairedDevices, device)
		} else {
			unpairedDevices = append(unpairedDevices, device)
		}
	}

	return pairedDevices, unpairedDevices
}

func RemoveDeviceByPath(devices []Device, path dbus.ObjectPath) []Device {
	for i, device := range devices {
		if device.Path == path {
			return append(devices[:i], devices[i+1:]...)
		}
	}
	return devices
}

// SortDevicesByRSSI sorts a slice of devices by RSSI in descending order.
func SortDevicesByRSSI(devices []Device) {
	slices.SortFunc(devices, func(a, b Device) int {
		// Primary sort: RSSI descending (higher is better)
		if a.RSSI > b.RSSI {
			return -1
		}
		if a.RSSI < b.RSSI {
			return 1
		}
		// Secondary sort: Name ascending (case-insensitive)
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
}
