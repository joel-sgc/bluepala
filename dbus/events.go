// dbus/events.go
package dbus

import (
	"bluepala/bluetooth" // Our new bluetooth package
	"bluepala/common"    // Our new types
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// WaitForDBusSignal is a tea.Cmd that blocks and waits for a single D-Bus signal.
// It translates the signal into a specific, granular tea.Msg for our Update loop.
// This is far more efficient than re-scanning all devices on every signal.
func WaitForDBusSignal(conn *dbus.Conn, sig chan *dbus.Signal) tea.Cmd {
	return func() tea.Msg {
		s := <-sig // Block until a signal is received

		switch s.Name {
		// --- ObjectManager Signals ---
		// These signals tell us when devices are added or removed.

		case bluetooth.ObjectManagerIF + ".InterfacesAdded":
			// A new device has appeared (e.g., from a scan)
			if len(s.Body) < 2 {
				break // Invalid signal
			}
			path, ok := s.Body[0].(dbus.ObjectPath)
			if !ok {
				break
			}
			// The interfaces map is the exact same format as GetManagedObjects
			interfaces, ok := s.Body[1].(map[string]map[string]dbus.Variant)
			if !ok {
				break
			}
			// Send a message with just the *new* device's data
			return common.DeviceAddedMsg{
				Path:       path,
				Interfaces: interfaces,
			}

		case bluetooth.ObjectManagerIF + ".InterfacesRemoved":
			// A device has been removed (e.g., "forgotten" or disconnected)
			if len(s.Body) < 1 {
				break // Invalid signal
			}
			path, ok := s.Body[0].(dbus.ObjectPath)
			if !ok {
				break
			}
			// Send a message telling the model to *remove* this one device
			return common.DeviceRemovedMsg{Path: path}

		// --- PropertiesChanged Signal ---
		// This signal tells us when a property on an *existing* device changes.

		case bluetooth.PropsIF + ".PropertiesChanged":
			// e.g., "Connected" changed from false to true
			if len(s.Body) < 2 {
				break // Invalid signal
			}
			iface, ok := s.Body[0].(string)
			if !ok {
				break
			}
			changes, ok := s.Body[1].(map[string]dbus.Variant)
			if !ok || len(changes) == 0 {
				break // No actual changes
			}

			// We only care about changes to Adapters, Devices, or Batteries
			if iface == bluetooth.AdapterIF || iface == bluetooth.DeviceIF || iface == bluetooth.BatteryIF {
				// Send a message with *only* the properties that changed
				return common.DevicePropertiesChangedMsg{
					Path:    s.Path, // The signal is emitted on the object that changed
					Changes: changes,
				}
			}
		}

		// If we didn't handle the signal, listen for the next one.
		return WaitForDBusSignal(conn, sig)()
	}
}

// RefreshAllDataCmd is a tea.Cmd that triggers a full, fresh state load.
// This is the equivalent of your old RefreshAllData.
func RefreshAllDataCmd(conn *dbus.Conn) tea.Cmd {
	return GetInitialStateCmd(conn) // GetInitialStateCmd already does this
}

// RefreshTicker returns a command that sends a PeriodicRefreshMsg
// on a 15-second interval, just like your old project.
func RefreshTicker() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		// In our main Update loop, this msg will trigger RefreshAllDataCmd
		return common.PeriodicRefreshMsg{}
	})
}