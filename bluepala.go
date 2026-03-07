// main.go
package main

import (
	"bluepala/bluetooth"
	"bluepala/common"
	"bluepala/config"
	"bluepala/dbus"
	"bluepala/models"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	godbus "github.com/godbus/dbus/v5"
	overlay "github.com/rmhubbert/bubbletea-overlay"
	"go.dalton.dog/bubbleup"
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
	Colors        config.Colors
	KeyMap        config.AppKeyMap
	Alert         bubbleup.AlertModel

	ConfirmationModal models.Confirmation
	IsModalActive     bool

	PairConfirmForm     models.PairConfirmForm
	IsPairConfirmActive bool

	RenameForm     models.RenameForm
	IsRenameActive bool
	RenameTarget   *common.Device

	Adapters        []common.Adapter
	PairedDevices   []common.Device
	UnpairedDevices []common.Device

	AdapterTable *models.TableData
	DevicesTable *models.TableData
	DetailsTable *models.TableData
	ScannedTable *models.TableData

	StatusBar *models.StatusBarData
}

func bluepalaModel() *BluepalaData {
	conn, err := godbus.SystemBus()
	if err != nil {
		cfg, _ := config.Load()
		return &BluepalaData{
			Err:    fmt.Errorf("failed to connect to D-Bus: %w", err),
			Colors: cfg.Colors,
		}
	}

	sigChan := make(chan *godbus.Signal, 10)
	conn.Signal(sigChan)

	appAgent := bluetooth.NewAgent()
	updateChan := make(chan tea.Msg)
	appAgent.SetUpdateChan(updateChan)

	cfg, _ := config.Load()
	colors := cfg.Colors
	keyMap := config.NewAppKeyMap(cfg)

	alert := bubbleup.NewAlertModel(40, true, 10)

	return &BluepalaData{
		Agent:           appAgent,
		UpdateChan:      updateChan,
		Conn:            conn,
		Err:             err,
		DBusSignals:     sigChan,
		SelectedTable:   1,
		PairedDevices:   make([]common.Device, 0),
		UnpairedDevices: make([]common.Device, 0),
		IsScanning:      false,
		Colors:          colors,
		KeyMap:          keyMap,
		Alert:           *alert,

		ConfirmationModal: models.ModelConfirmation(colors),
		IsModalActive:     false,

		PairConfirmForm:     models.ModelPairConfirmForm(colors),
		IsPairConfirmActive: false,

		RenameForm:     models.ModelRenameForm(colors),
		IsRenameActive: false,
		RenameTarget:   nil,

		DevicesTable: &models.TableData{
			Conn:            conn,
			IsTableSelected: true,
			Title:           "Devices",
			Height:          11,
			PairedDevices:   make([]common.Device, 0),
			Colors:          colors,
		},
		DetailsTable: &models.TableData{
			Title:           "Details",
			IsTableSelected: true,
			Height:          12,
			Width:           30,
			Colors:          colors,
		},
		ScannedTable: &models.TableData{
			Conn:           conn,
			Title:          "Nearby Devices",
			Height:         16,
			ScannedDevices: make([]common.Device, 0),
			Colors:         colors,
		},
		AdapterTable: &models.TableData{
			Conn:            conn,
			Title:           "Adapter",
			IsTableSelected: false,
			Height:          5,
			Colors:          colors,
		},

		StatusBar: &models.StatusBarData{
			KeyMap: keyMap,
			Colors: colors,
		},
	}
}

func (m BluepalaData) Sub() tea.Cmd {
	return func() tea.Msg {
		return <-m.UpdateChan
	}
}

func (m *BluepalaData) Init() tea.Cmd {
	if m.Err != nil {
		return nil
	}
	return tea.Batch(
		m.Alert.Init(),
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

	if m.IsPairConfirmActive {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height
		case common.SubmitConfirmMsg:
			m.Agent.SubmitConfirmation(msg.Confirmed)
			m.IsPairConfirmActive = false
			// Re-start signal listener and refresh device list after pairing completes.
			return m, tea.Batch(
				dbus.WaitForDBusSignal(m.Conn, m.DBusSignals),
				dbus.GetInitialStateCmd(m.Conn),
			)
		default:
			var formCmd tea.Cmd
			var updatedForm tea.Model
			updatedForm, formCmd = m.PairConfirmForm.Update(msg)
			m.PairConfirmForm = updatedForm.(models.PairConfirmForm)
			// Re-queue signal listener if a D-Bus signal message arrived while modal was open.
			switch msg.(type) {
			case common.DevicePropertiesChangedMsg, common.DeviceAddedMsg,
				common.DeviceRemovedMsg, common.AdapterPropertiesChangedMsg,
				common.DeviceUpdateMsg, common.AdapterUpdateMsg:
				return m, tea.Batch(formCmd, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))
			}
			return m, formCmd
		}
	}

	if m.IsRenameActive {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height
		case common.SubmitRenameMsg:
			m.IsRenameActive = false
			if msg.Name != "" && m.RenameTarget != nil {
				return m, dbus.RenameDeviceCmd(m.Conn, m.RenameTarget.Path, msg.Name)
			}
			return m, nil
		default:
			var formCmd tea.Cmd
			var updatedForm tea.Model
			updatedForm, formCmd = m.RenameForm.Update(msg)
			m.RenameForm = updatedForm.(models.RenameForm)
			return m, formCmd
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
		// Look up the device by path so we can show its info
		var device *common.Device
		for i := range m.UnpairedDevices {
			if m.UnpairedDevices[i].Path == msg.DevicePath {
				device = &m.UnpairedDevices[i]
				break
			}
		}
		if device == nil {
			for i := range m.PairedDevices {
				if m.PairedDevices[i].Path == msg.DevicePath {
					device = &m.PairedDevices[i]
					break
				}
			}
		}
		m.PairConfirmForm.Device = device
		m.PairConfirmForm.Passkey = msg.Passkey
		m.PairConfirmForm.ConfirmValue = false
		m.IsPairConfirmActive = true
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

		// Sort only when the list structurally changes, not on every RSSI update.
		common.SortDevicesByRSSI(m.UnpairedDevices)

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

		common.SortDevicesByRSSI(m.UnpairedDevices)

		cmds = append(cmds, dbus.WaitForDBusSignal(m.Conn, m.DBusSignals))

	case common.DeviceUpdateMsg:
		// This message comes on startup with the full list of devices.
		// We just need to filter and split them.
		paired, unpaired := common.FilterDevicesByPaired(msg)
		m.PairedDevices = paired
		m.UnpairedDevices = unpaired

		// Sort unpaired devices on full list replacement.
		common.SortDevicesByRSSI(m.UnpairedDevices)

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
		alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Error: "+msg.Err.Error())
		return m, alertCmd

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

		switch {
		case key.Matches(msg, m.KeyMap.Select):
			switch m.SelectedTable {
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
				case 2:
					// Rename device
					m.RenameForm.Input.SetValue("")
					m.RenameForm.Input.Focus()
					m.RenameForm.Device = device
					m.RenameForm.ConfirmValue = false
					m.RenameTarget = device
					m.IsRenameActive = true
					return m, m.RenameForm.Init()
				}
			case 2:
				if len(m.UnpairedDevices) > m.ScannedTable.SelectedRow {
					device := &m.UnpairedDevices[m.ScannedTable.SelectedRow]
					cmd := dbus.PairDeviceCmd(m.Conn, device.Path)
					cmds = append(cmds, cmd)
				}
			case 0:
				// Toggle adapter power
				adapter := &m.Adapters[m.AdapterTable.SelectedRow]
				adapter.Powered = !adapter.Powered
				dbus.ToggleAdapterPowerCmd(m.Conn, string(adapter.Path), !adapter.Powered)
			}

		case key.Matches(msg, m.KeyMap.Quit):
			m.Conn.RemoveSignal(m.DBusSignals)
			m.Conn.Close()
			return m, tea.Quit

		case key.Matches(msg, m.KeyMap.NextPane), key.Matches(msg, m.KeyMap.PrevPane):
			tables := []*models.TableData{
				m.AdapterTable,
				m.DevicesTable,
				m.ScannedTable,
			}

			if key.Matches(msg, m.KeyMap.NextPane) {
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

		case key.Matches(msg, m.KeyMap.Scan):
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

	// Always ensure tables have the latest data before rendering
	m.AdapterTable.Adapters = m.Adapters
	m.DevicesTable.PairedDevices = m.PairedDevices
	m.ScannedTable.ScannedDevices = m.UnpairedDevices

	// Pass every message through the alert model so it can tick and dismiss.
	var updatedAlert tea.Model
	var alertCmd tea.Cmd
	updatedAlert, alertCmd = m.Alert.Update(msg)
	m.Alert = updatedAlert.(bubbleup.AlertModel)
	cmds = append(cmds, alertCmd)

	return m, tea.Batch(cmds...)
}

func (m *BluepalaData) View() string {
	// Fatal startup error (e.g. D-Bus unavailable) — show full-screen error.
	if m.Conn == nil {
		return models.ModelError(
			"Failed to connect to D-Bus.\n\n"+m.Err.Error()+"\n\nPress Enter, Esc, or Ctrl+C to quit.",
			m.Colors,
		).View()
	}

	if m.IsModalActive {
		bgModel := backgroundModel{m}
		fgModel := &m.ConfirmationModal

		overlayModel := overlay.New(fgModel, bgModel, overlay.Left, overlay.Top, 0, 0)
		return m.Alert.Render(overlayModel.View())
	}

	if m.IsPairConfirmActive {
		bgModel := backgroundModel{m}
		fgModel := &m.PairConfirmForm

		overlayModel := overlay.New(fgModel, bgModel, overlay.Left, overlay.Center, common.CalculatePadding(fgModel.View()), 0)
		return m.Alert.Render(overlayModel.View())
	}

	if m.IsRenameActive {
		bgModel := backgroundModel{m}
		fgModel := &m.RenameForm

		overlayModel := overlay.New(fgModel, bgModel, overlay.Left, overlay.Center, common.CalculatePadding(fgModel.View()), 0)
		return m.Alert.Render(overlayModel.View())
	}

	return m.Alert.Render(m.MainView())
}

func main() {
	p := tea.NewProgram(bluepalaModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
