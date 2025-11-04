// main.go
package main

import (
	"bluepala/bluetooth"
	"bluepala/common"
	"bluepala/dbus"
	"bluepala/models"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	godbus "github.com/godbus/dbus/v5"
)

type BluepalaData struct {
	DBusSignals 	chan *godbus.Signal
	Conn        	*godbus.Conn
	Err         	error
	Width, Height int
	SelectedTable int
	IsScanning    bool

	Adapters 				[]common.Adapter
	PairedDevices  	[]common.Device
	UnpairedDevices []common.Device

	AdapterTable *models.TableData
	DevicesTable *models.TableData
	DetailsTable *models.TableData
	ScannedTable *models.TableData
}

func bluepalaModel() *BluepalaData {
	conn, err := godbus.SystemBus()
	if err != nil {
		return &BluepalaData{Err: fmt.Errorf("failed to connect to D-Bus: %w", err)}
	}

	sigChan := make(chan *godbus.Signal, 10)
	conn.Signal(sigChan)

	return &BluepalaData{
		Conn:        	conn,
		Err:         	err,
		DBusSignals: 	sigChan,
		SelectedTable: 0,
		PairedDevices:   make([]common.Device, 0),
    UnpairedDevices: make([]common.Device, 0),
		IsScanning: false,
		
		AdapterTable:	&models.TableData{Conn: conn, Title: "Adapter", IsTableSelected: true},
		DevicesTable:	&models.TableData{
			Conn: conn, 
			Title: "Devices",
			Height: 8,
			PairedDevices:   make([]common.Device, 0),
		},
		DetailsTable:	&models.TableData{
			Title: "Details", 
			Height: 12,
			Width: 30,
		},
		ScannedTable:	&models.TableData{
			Conn: conn, 
			Title: "Nearby Devices",
			Height: 14,
			ScannedDevices: make([]common.Device, 0),
		},
	}
}

func (m *BluepalaData) Init() tea.Cmd {
	return tea.Batch(
		dbus.GetInitialStateCmd(m.Conn),
		dbus.RefreshTicker(),
		dbus.WaitForDBusSignal(m.Conn, m.DBusSignals),
	)
}

func (m *BluepalaData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Forward to submodel (pointer-based)
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		
		m.AdapterTable.Width = msg.Width
		m.DevicesTable.Width = msg.Width - 36
		m.DetailsTable.Width = 36
		m.ScannedTable.Width = msg.Width

	case common.PeriodicRefreshMsg:
		cmd := dbus.GetInitialStateCmd(m.Conn)
		cmds = append(cmds, cmd)

	case common.DevicePropertiesChangedMsg:
		updateDevice := func(devices []common.Device) {
			for i := range devices {
				device := &devices[i]
				if device.Path == msg.Path {
					if nameVariant, ok := msg.Changes["Name"]; ok {
						device.Name = nameVariant.Value().(string)
					}
					if aliasVariant, ok := msg.Changes["Alias"]; ok {
						device.Name = aliasVariant.Value().(string)
					}
					
					if icon, ok := msg.Changes["Icon"]; ok {
						device.Icon = bluetooth.NormalizeIcon(icon.String())
					}
					if paired, ok := msg.Changes["Paired"]; ok {
						device.Paired, _ = paired.Value().(bool)
					}
					if trusted, ok := msg.Changes["Trusted"]; ok {
						device.Trusted, _ = trusted.Value().(bool)
					}
					if connected, ok := msg.Changes["Connected"]; ok {
						device.Connected, _ = connected.Value().(bool)
					}
					if connectable, ok := msg.Changes["Connectable"]; ok {
						device.Connectable, _ = connectable.Value().(bool)
					}
					if rssi, ok := msg.Changes["RSSI"]; ok {
						device.RSSI = rssi.Value().(int16)
					}
					if battery, ok := msg.Changes["Percentage"]; ok {
						device.Battery = int8(battery.Value().(byte))
					}
				}
			}
		}
		updateDevice(m.PairedDevices)
		updateDevice(m.UnpairedDevices)
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.AdapterUpdateMsg:
		m.Adapters = msg
		_, cmd := m.AdapterTable.Update(msg)
		cmds = append(cmds, cmd)

	case common.DeviceAddedMsg:
		// 1. Check if the new object is a Device.
    // (The signal could be for a GATT service, which we ignore)
    deviceProps, isDevice := msg.Interfaces[bluetooth.DeviceIF]
    if !isDevice {
			return m, nil // Not a device, ignore it.
    }

    // 2. Check if we already have this device (just in case)
		allDevices := append(m.PairedDevices, m.UnpairedDevices...)
    for _, dev := range allDevices {
			if dev.Path == msg.Path {
				return m, nil // Already in our list.
			}
    }

    // 3. It's a new device! Parse it.
    // Check for battery info (it might be nil)
    batteryProps := msg.Interfaces[bluetooth.BatteryIF]

    // Use our new public parser function:
    newDevice := bluetooth.ParseDevice(msg.Path, deviceProps, batteryProps)

    // 4. Add the new device to our main list
		if newDevice.Paired {
			m.PairedDevices = append(m.PairedDevices, newDevice)

			if len(m.PairedDevices) > 0 {
				m.DetailsTable.SelectedPaired = &m.PairedDevices[0]
			} else {
				m.DetailsTable.SelectedPaired = &common.BlankDevice
			}

			m.DevicesTable.PairedDevices = m.PairedDevices
		} else {
			m.UnpairedDevices = append(m.UnpairedDevices, newDevice)
			m.ScannedTable.ScannedDevices = m.UnpairedDevices
		}

		_, cmd := m.DevicesTable.Update(msg); cmds = append(cmds, cmd)
		_, cmd = m.ScannedTable.Update(msg); cmds = append(cmds, cmd)
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DeviceRemovedMsg:
		removeDevice := func(devices []common.Device) []common.Device {
			for i, device := range devices {
				if device.Path == msg.Path {
					return append(devices[:i], devices[i+1:]...)
				}
			}
			return devices
		}
		m.PairedDevices = removeDevice(m.PairedDevices)
		m.UnpairedDevices = removeDevice(m.UnpairedDevices)

		_, cmd := m.DevicesTable.Update(msg); cmds = append(cmds, cmd)
		_, cmd = m.ScannedTable.Update(msg); cmds = append(cmds, cmd)
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DeviceUpdateMsg:
		paired, unpaired := filterDevicesByPaired(msg)

		m.PairedDevices = paired
		m.UnpairedDevices = unpaired

		if m.PairedDevices == nil {
			m.PairedDevices = make([]common.Device, 0)
		}
		if m.UnpairedDevices == nil {
			m.UnpairedDevices = make([]common.Device, 0)
		}

		_, cmd := m.DevicesTable.Update(msg); cmds = append(cmds, cmd)
		_, cmd = m.ScannedTable.Update(msg); cmds = append(cmds, cmd)

		cmd = func() tea.Cmd {
			return func() tea.Msg {
				if len(m.PairedDevices) > 0 {
					return common.DeviceSelectedMsg{Device: m.PairedDevices[0]}
				} else {
					return common.DeviceSelectedMsg{Device: common.BlankDevice}
				}
			}
		}()

		cmds = append(cmds, cmd)
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DeviceSelectedMsg:
		_, cmd := m.DetailsTable.Update(msg); cmds = append(cmds, cmd)
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.ErrMsg:
		m.Err = msg.Err

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			switch m.SelectedTable {
			case 0:
				// Toggle adapter power
				adapter := &m.Adapters[m.AdapterTable.SelectedRow]
				adapter.Powered = !adapter.Powered
				dbus.ToggleAdapterPowerCmd(m.Conn, string(adapter.Path), !adapter.Powered)				
			case 1:
				device := &m.PairedDevices[m.DevicesTable.SelectedRow]
				if (device.Path == "-1" || len(m.PairedDevices) <= 0) {
					// Do nothing for the blank device
					return m, nil
				}

				switch m.DetailsTable.SelectedRow {
				case 0:
					// Toggle paired device connection
					cmd := dbus.ConnectToDeviceCmd(m.Conn, m.PairedDevices, device, !device.Connected)
					cmds = append(cmds, cmd)
				case 1:
					// Forget device
					cmd := dbus.ForgetDeviceCmd(m.Conn, m.Adapters[0].Path, device.Path)
					cmds = append(cmds, cmd)
				}
			}
		case "ctrl+c", "ctrl+q":
			m.Conn.RemoveSignal(m.DBusSignals)
			m.Conn.Close()
			return m, tea.Quit
		case "tab", "shift+tab":
			tables := []*models.TableData{
				m.AdapterTable,
				m.DevicesTable,
				m.ScannedTable,
			}

			if msg.String() == "tab" {
				m.SelectedTable = (m.SelectedTable + 1) % len(tables)
			} else {
				m.SelectedTable = (m.SelectedTable - 1 + len(tables)) % len(tables)
			}

			for i, table := range tables {
				// Deselect all tables
				table.IsTableSelected = false

				// Select the current table
				if i == m.SelectedTable {
					table.IsTableSelected = true
					m.DetailsTable.IsTableSelected = i == 1
				}
			}
		case "left", "h", "right", "l":
			if m.SelectedTable == 1 {
				_, cmd := m.DetailsTable.Update(msg)
				cmds = append(cmds, cmd)
			}
		case "up", "k", "down", "j":
			tables := []*models.TableData{
				m.AdapterTable,
				m.DevicesTable,
				m.DetailsTable,
				m.ScannedTable,
			}

			_, cmd := tables[m.SelectedTable].Update(msg)
			cmds = append(cmds, cmd)
		case "s":
			if (!m.IsScanning) {
				cmds = append(cmds, dbus.StartScanning(m.Conn, &m.Adapters[0].Path))
			} else {
				cmds = append(cmds, dbus.StopScanning(m.Conn, &m.Adapters[0].Path))
			}
		}

		_, cmd := m.DevicesTable.Update(msg); cmds = append(cmds, cmd)
		_, cmd = m.ScannedTable.Update(msg); cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *BluepalaData) View() string {
	if m.Err != nil {
		return fmt.Sprintf("An error occurred: %v\n\nPress 'ctrl+q' to quit.", m.Err)
	}

	return lipgloss.JoinVertical(lipgloss.Left, 
		m.AdapterTable.View(),
		strings.TrimSpace(common.HJoin(
			m.DevicesTable.View(),
			m.DetailsTable.View(),
			m.DevicesTable.Width,
			m.DetailsTable.Width,
		)),
		m.ScannedTable.View(),
	)
}

func main() {
	p := tea.NewProgram(bluepalaModel(), tea.WithAltScreen(), tea.WithoutCatchPanics())
	if _, err := p.Run(); err != nil {
		fmt.Println("Program exited with error:", err)
		os.Exit(1)
	}
}

// Returns two slices: paired devices and unpaired devices
func filterDevicesByPaired(devices []common.Device) ([]common.Device, []common.Device) {
	pairedDevices := make([]common.Device, 0)
	unpairedDevices := make([]common.Device, 0)

	for _, device := range devices {
		if device.Paired {
			pairedDevices = append(pairedDevices, device)
		} else {
			unpairedDevices = append(unpairedDevices, device)
		}
	}

	return pairedDevices, unpairedDevices
}