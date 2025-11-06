// dbus/actions.go
package dbus

import (
	"bluepala/bluetooth"
	"bluepala/common"
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// GetInitialStateCmd is a tea.Cmd that fetches the initial BlueZ state.
// It calls our synchronous GetInitialState function and returns
// the results as messages for the BubbleTea update loop.
func GetInitialStateCmd(conn *dbus.Conn) tea.Cmd {
	return func() tea.Msg {
		adapters, devices, err := bluetooth.GetInitialState(conn)
		if err != nil {
			return common.ErrMsg{Err: err}
		}

		// Send both lists back in a batch
		// We'll update common/types.go to include these messages
		return tea.Batch(
			func() tea.Msg { return common.AdapterUpdateMsg(adapters) },
			func() tea.Msg { return common.DeviceUpdateMsg(devices) },
			func() tea.Msg { return common.PeriodicRefreshMsg{} },
		)()
	}
}

func ToggleAdapterPowerCmd(conn *dbus.Conn, adapterPath string, state bool) error {
	// First, ensure rfkill doesn't block us
	rfkillState := "unblock"
	if state {
		rfkillState = "block"
	}
	
	cmd := exec.Command("rfkill", rfkillState, "bluetooth")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to %s bluetooth via rfkill: %v", rfkillState, err)
	}

	// Now set the Bluez power state
	obj := conn.Object(bluetooth.BluezDest, dbus.ObjectPath(adapterPath))
	call := obj.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		bluetooth.AdapterIF,
		"Powered", 
		dbus.MakeVariant(state),
	)
	
	return call.Err
}

// Non-blocking connection command
func ConnectToDeviceCmd(conn *dbus.Conn, devices []common.Device, device *common.Device, status bool) tea.Cmd {
	return func() tea.Msg {
		path := dbus.ObjectPath(device.Path)

		action := "Disconnect"
		if status {
			action = "Connect"
		}

		obj := conn.Object(bluetooth.BluezDest, dbus.ObjectPath(path))
		call := obj.Call(fmt.Sprintf("%s.%s", bluetooth.DeviceIF, action), 0)

		if call.Err != nil {
			return common.ErrMsg{Err: call.Err}
		}

		// On success, we don't need to return anything.
		// BlueZ will emit a "PropertiesChanged" signal for the "Connected" property,
		// which our main event loop will catch and use to update the UI.
		return nil
	}
}

func ForgetDeviceCmd(conn *dbus.Conn, adapterPath dbus.ObjectPath, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		// 1. Get the adapter object (e.g., /org/bluez/hci0)
		adapterObj := conn.Object(bluetooth.BluezDest, adapterPath)

		// 2. Call the RemoveDevice method on the adapter
		call := adapterObj.Call(
			bluetooth.AdapterIF+".RemoveDevice", // Method name
			0,                                 // Flags
			devicePath,                        // Argument 1: path of device to remove
		)
		
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to remove device %s: %w", devicePath, call.Err)}
		}

		// 3. Success!
		// We return 'nil' because BlueZ will now emit an
		// 'InterfacesRemoved' signal. Our dbus/events.go
		// listener will catch this and send a
		// 'common.DeviceRemovedMsg' to our Update loop,
		// which will then remove the device from the list.
		return nil
	}
}

func PairDeviceCmd(conn *dbus.Conn, devicePath dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		// 1. Get the device object
		deviceObj := conn.Object(bluetooth.BluezDest, devicePath)

		// 2. Call the Pair method
		call := deviceObj.Call(
			bluetooth.DeviceIF+".Pair", // Method name
			0,                          // Flags
		)
		
		if call.Err != nil {
			// This will often return 'AuthenticationFailed' or 'AuthenticationCanceled'
			// if no Agent is available to handle a PIN request.
			return common.ErrMsg{Err: fmt.Errorf("failed to pair with %s: %w", devicePath, call.Err)}
		}

		// 3. Success!
		// We return 'nil'. The pairing process is now active.
		// Our 'WaitForDBusSignal' listener will eventually
		// receive a 'PropertiesChanged' signal for 'Paired = true'
		// if the pairing succeeds.
		return nil
	}
}

func StartScanning(conn *dbus.Conn, adapterPath *dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		obj := conn.Object(bluetooth.BluezDest, *adapterPath)
		call := obj.Call(bluetooth.AdapterIF+".StartDiscovery", 0)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to start discovery: %v", call.Err)}
		}
		return common.ScanToggleMsg{}
	}
}

// StopScanning stops Bluetooth device discovery
func StopScanning(conn *dbus.Conn, adapterPath *dbus.ObjectPath) tea.Cmd {
	return func() tea.Msg {
		obj := conn.Object(bluetooth.BluezDest, *adapterPath)
		call := obj.Call(bluetooth.AdapterIF+".StopDiscovery", 0)
		if call.Err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to stop discovery: %v", call.Err)}
		}
		return common.ScanToggleMsg{}
	}
}