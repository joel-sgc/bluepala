// main.go
package main

import (
	"bluepala/bluetooth"
	"bluepala/common"
	"bluepala/dbus"
	"bluepala/models"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	godbus "github.com/godbus/dbus/v5"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

type BluepalaData struct {
	Agent         *bluetooth.BluepalaAgent
	UpdateChan    chan tea.Msg
	DBusSignals   chan *godbus.Signal
	Conn          *godbus.Conn
	Err           error
	Width, Height int
	SelectedTable int
	IsScanning    bool

	ConfirmationModal models.Confirmation
	IsModalActive     bool

	Adapters        []common.Adapter
	PairedDevices   []common.Device
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

	appAgent := bluetooth.NewAgent()
	updateChan := make(chan tea.Msg)

	// Give the agent the channel so it can send messages
	appAgent.SetUpdateChan(updateChan)

	return &BluepalaData{
		Agent:           appAgent,
		UpdateChan:      updateChan,
		Conn:            conn,
		Err:             err,
		DBusSignals:     sigChan,
		SelectedTable:   0,
		PairedDevices:   make([]common.Device, 0),
		UnpairedDevices: make([]common.Device, 0),
		IsScanning:      false,

		ConfirmationModal: models.ModelConfirmation(),
		IsModalActive:     false,

		AdapterTable: &models.TableData{Conn: conn, Title: "Adapter", IsTableSelected: true},
		DevicesTable: &models.TableData{
			Conn:          conn,
			Title:         "Devices",
			Height:        11,
			PairedDevices: make([]common.Device, 0),
		},
		DetailsTable: &models.TableData{
			Title:  "Details",
			Height: 12,
			Width:  30,
		},
		ScannedTable: &models.TableData{
			Conn:           conn,
			Title:          "Nearby Devices",
			Height:         15,
			ScannedDevices: make([]common.Device, 0),
		},
	}
}

func (m BluepalaData) Sub() tea.Cmd {
	return func() tea.Msg {
		return <-m.UpdateChan
	}
}

func (m *BluepalaData) Init() tea.Cmd {
	return tea.Batch(
		dbus.RegisterAgentCmd(m.Conn, m.Agent), // Register the agent on startup
		dbus.GetInitialStateCmd(m.Conn),
		dbus.RefreshTicker(),
		dbus.WaitForDBusSignal(m.Conn, m.DBusSignals),
		m.Sub(), // Start listening for agent messages
	)
}

func (m *BluepalaData) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.IsModalActive {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height

			m.AdapterTable.Width = msg.Width
			m.DevicesTable.Width = msg.Width - 36
			m.DetailsTable.Width = 36
			m.ScannedTable.Width = msg.Width

		case common.SubmitConfirmMsg:
			m.Agent.SubmitConfirmation(msg.Confirmed)
			m.IsModalActive = false
			// The agent is now unblocked, no further command needed here.
			return m, nil
		default:
			var modalCmd tea.Cmd
			var updatedModal tea.Model
			updatedModal, modalCmd = m.ConfirmationModal.Update(msg)
			m.ConfirmationModal = updatedModal.(models.Confirmation)
			// We don't re-subscribe. The subscription from Init is still active.
			return m, modalCmd
		}
	}

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

	// --- AGENT MESSAGES ---
	case common.ShowPinModalMsg:
		// A PIN is required!
		// 1. Set model state to show modal
		// m.modal = common.NewPinModal(msg.DevicePath)
		// 2. Return the command to focus the modal
		// return m, m.modal.Focus()
		log.Printf("TUI: ShowPinModalMsg received! Device: %s", msg.DevicePath)

	case common.ShowConfirmModalMsg:
		m.IsModalActive = true
		m.ConfirmationModal.Message = "Confirm pairing to " + msg.DeviceName + "?"
		m.ConfirmationModal.Value = false
		// No need to re-subscribe.
		return m, nil

	case common.SubmitPinMsg:
		// The modal is sending us a PIN
		// 1. Send the PIN to the agent (which unblocks it)
		m.Agent.SubmitPin(msg.Pin)
		// 2. Close the modal
		// m.modal = nil
		log.Printf("TUI: Sending PIN '%s' to agent", msg.Pin)

	case common.SubmitConfirmMsg:
		// This case is now handled by the IsModalActive block at the top.
		// We can leave this here as a fallback if needed, but it shouldn't be hit.
		m.Agent.SubmitConfirmation(msg.Confirmed)
		m.IsModalActive = false
		log.Printf("TUI: Sending Confirmation '%t' to agent", msg.Confirmed)

	case common.AdapterPropertiesChangedMsg:
		for i := range m.Adapters {
			adapter := &m.Adapters[i]
			if adapter.Path == msg.Path {
				if discovering, ok := msg.Changes["Discovering"]; ok {
					adapter.Scanning, _ = discovering.Value().(bool)
				}
			}
		}
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DevicePropertiesChangedMsg:
		var newlyPairedDevice *common.Device
		var newlyUnpairedDevice *common.Device
		var deviceToConnect *common.Device

		// This helper function now has access to the 'cmds' slice
		updateDevice := func(devices []common.Device, isPairedList bool) {
			for i := range devices {
				device := &devices[i]
				// --- Find the device that changed ---
				if device.Path == msg.Path {

					// --- Check for pairing change FIRST ---
					if paired, ok := msg.Changes["Paired"]; ok {
						if newPairedStatus, ok := paired.Value().(bool); ok && newPairedStatus != device.Paired {
							// Pairing status changed!
							device.Paired = newPairedStatus

							if newPairedStatus && !isPairedList {
								// Device just got PAIRED
								newlyPairedDevice = device
							} else if !newPairedStatus && isPairedList {
								// Device just got UN-PAIRED
								newlyUnpairedDevice = device
							}
						}
					}

					// --- Check for trust change ---
					if trusted, ok := msg.Changes["Trusted"]; ok {
						if newTrustedStatus, ok := trusted.Value().(bool); ok && newTrustedStatus && !device.Trusted {
							// Device just became trusted. If it's paired and not connected, we should connect.
							if device.Paired && !device.Connected {
								deviceToConnect = device
							}
						}
					}

					// --- Apply all other property changes ---
					if nameVariant, ok := msg.Changes["Name"]; ok {
						if name, ok := nameVariant.Value().(string); ok {
							device.Name = name
						}
					}
					if aliasVariant, ok := msg.Changes["Alias"]; ok {
						// Alias (user-set name) always wins
						if alias, ok := aliasVariant.Value().(string); ok {
							device.Name = alias
						}
					}
					if icon, ok := msg.Changes["Icon"]; ok {
						if iconStr, ok := icon.Value().(string); ok {
							device.Icon = bluetooth.NormalizeIcon(iconStr)
						}
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
						device.RSSI, _ = rssi.Value().(int16)
					}
					if battery, ok := msg.Changes["Percentage"]; ok {
						if val, ok := battery.Value().(byte); ok {
							device.Battery = int8(val)
						}
					}

					// We found and updated the device, no need to keep looping
					return
				}
			}
		}

		// Run the update logic on both lists
		updateDevice(m.PairedDevices, true)
		updateDevice(m.UnpairedDevices, false)

		// --- Move device if pairing status changed ---
		if newlyPairedDevice != nil {
			m.UnpairedDevices = common.RemoveDeviceByPath(m.UnpairedDevices, newlyPairedDevice.Path)
			m.PairedDevices = append(m.PairedDevices, *newlyPairedDevice)

			// The device was just paired, so we MUST trust it.
			// The connection command will be sent later, after we get the "Trusted" signal.
			cmds = append(cmds, dbus.TrustDeviceCmd(m.Conn, newlyPairedDevice.Path))
		}

		if newlyUnpairedDevice != nil {
			m.PairedDevices = common.RemoveDeviceByPath(m.PairedDevices, newlyUnpairedDevice.Path)
			m.UnpairedDevices = append(m.UnpairedDevices, *newlyUnpairedDevice)
		}

		// --- Connect if a device just became trusted ---
		if deviceToConnect != nil {
			cmds = append(cmds, dbus.ConnectToDeviceCmd(m.Conn, nil, deviceToConnect, true))
		}

		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.AdapterUpdateMsg:
		m.Adapters = msg
		// The table doesn't need to process this, just have the data for viewing
		m.AdapterTable.Adapters = m.Adapters

	case common.DeviceAddedMsg:
		// 1. Check if the new object is a usable Device.
		if !bluetooth.IsUsableDevice(msg.Interfaces) {
			return m, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals)
		}
		deviceProps := msg.Interfaces[bluetooth.DeviceIF]

		// 2. Parse the device from the message.
		batteryProps := msg.Interfaces[bluetooth.BatteryIF]
		newDevice := bluetooth.ParseDevice(msg.Path, deviceProps, batteryProps)

		// 3. Check if the device exists to update it, otherwise add it.
		found := false
		// Check paired devices
		for i, dev := range m.PairedDevices {
			if dev.Path == newDevice.Path {
				m.PairedDevices[i] = newDevice // Update existing device
				found = true
				break
			}
		}
		// Check unpaired devices if not found yet
		if !found {
			for i, dev := range m.UnpairedDevices {
				if dev.Path == newDevice.Path {
					m.UnpairedDevices[i] = newDevice // Update existing device
					found = true
					break
				}
			}
		}

		// 4. If not found anywhere, it's a new device. Add it to the correct list.
		if !found {
			if newDevice.Paired {
				m.PairedDevices = append(m.PairedDevices, newDevice)
			} else {
				m.UnpairedDevices = append(m.UnpairedDevices, newDevice)
			}
		}

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

		// If the removed device was the selected one, clear the details view
		if m.DetailsTable.SelectedPaired != nil && m.DetailsTable.SelectedPaired.Path == msg.Path {
			m.DetailsTable.SelectedPaired = nil
		}

		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DeviceUpdateMsg:
		// This message comes on startup with the full list of devices.
		// We just need to filter and split them.
		paired, unpaired := common.FilterDevicesByPaired(msg)
		m.PairedDevices = paired
		m.UnpairedDevices = unpaired

		// Automatically select the first paired device on startup, if it exists.
		cmd := func() tea.Msg {
			if len(m.PairedDevices) > 0 {
				return common.DeviceSelectedMsg{Device: &m.PairedDevices[0]}
			}
			return common.DeviceSelectedMsg{Device: nil} // Otherwise, select nothing.
		}
		cmds = append(cmds, cmd)
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DeviceSelectedMsg:
		m.DetailsTable.SelectedPaired = msg.Device
		// This is a UI update, no need to wait for another D-Bus signal.
		return m, nil

	case common.ScanToggleMsg:
		m.IsScanning = !m.IsScanning
		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.ErrMsg:
		m.Err = msg.Err

	case tea.KeyMsg:
		// Give the active table the key press.
		var cmd tea.Cmd
		switch m.SelectedTable {
		case 0: // Adapters
			_, cmd = m.AdapterTable.Update(msg)
		case 1: // Paired Devices
			_, cmd = m.DevicesTable.Update(msg)
			// Also forward horizontal movement to details table
			if msg.String() == "left" || msg.String() == "h" || msg.String() == "right" || msg.String() == "l" {
				_, cmd = m.DetailsTable.Update(msg)
			}
		case 2: // Scanned Devices
			_, cmd = m.ScannedTable.Update(msg)
		}
		cmds = append(cmds, cmd)

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
				if device.Path == "-1" || len(m.PairedDevices) <= 0 {
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
			case 2:
				if len(m.UnpairedDevices) > m.ScannedTable.SelectedRow {
					device := &m.UnpairedDevices[m.ScannedTable.SelectedRow]
					cmd := dbus.PairDeviceCmd(m.Conn, device.Path)
					cmds = append(cmds, cmd)
				}
			}

		case "ctrl+c", "ctrl+q", "q", "ctrl+w":
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
			// This is now handled above and forwarded to the details table
			// if the paired devices table is selected.
		case "up", "k", "down", "j":
			// This is now handled by the active table directly.
		case "s":
			if len(m.Adapters) > 0 {
				if !m.IsScanning {
					cmds = append(cmds, dbus.StartScanning(m.Conn, &m.Adapters[0].Path))
				} else {
					cmds = append(cmds, dbus.StopScanning(m.Conn, &m.Adapters[0].Path))
				}
			}
		}

		// After any action, give the tables the latest data to render.
		m.AdapterTable.Adapters = m.Adapters
		m.DevicesTable.PairedDevices = m.PairedDevices
		m.ScannedTable.ScannedDevices = m.UnpairedDevices
	}

	// Sort unpaired devices by RSSI to prevent flickering
	common.SortDevicesByRSSI(m.UnpairedDevices)

	// Always ensure tables have the latest data before rendering
	m.AdapterTable.Adapters = m.Adapters
	m.DevicesTable.PairedDevices = m.PairedDevices
	m.ScannedTable.ScannedDevices = m.UnpairedDevices

	return m, tea.Batch(cmds...)
}

func (m *BluepalaData) View() string {
	if m.Err != nil {
		return lipgloss.NewStyle().Width(common.WindowDimensions().Width).Render(
			"Program exited with error:" + m.Err.Error() + "\n\nPress 'ctrl+q' to quit.",
		)
	}

	if m.IsModalActive {
		bgModel := backgroundModel{m} // wrap main model
		fgModel := &m.ConfirmationModal

		overlayModel := overlay.New(fgModel, bgModel, overlay.Left, overlay.Top, 0, 0)
		return overlayModel.View()
	}

	return m.MainView()
}

func main() {
	p := tea.NewProgram(bluepalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
