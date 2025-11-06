package common

import (
	"github.com/godbus/dbus/v5"
)

// AdapterUpdateMsg sends a new list of adapters to the model
type AdapterUpdateMsg []Adapter
type PeriodicRefreshMsg []Adapter
type DeviceUpdateMsg []Device

// --- D-Bus Signal Messages ---
type AdapterPropertiesChangedMsg struct {
	Path    dbus.ObjectPath
	Changes map[string]dbus.Variant
}

type DevicePropertiesChangedMsg struct {
	Path    dbus.ObjectPath
	Changes map[string]dbus.Variant // Map of property name to new value
}

// DeviceSelectedMsg is sent when a device is selected for our details table
type DeviceSelectedMsg struct {
	Device *Device
}

// DeviceAddedMsg is sent when a new device is discovered
type DeviceAddedMsg struct {
	Path       dbus.ObjectPath
	Interfaces map[string]map[string]dbus.Variant
}

// DeviceRemovedMsg is sent when a device is removed
type DeviceRemovedMsg struct {
	Path dbus.ObjectPath
}

// ScanToggleMsg is sent when scanning is started or stopped
type ScanToggleMsg struct{}

// ErrMsg reports a generic error from a goroutine
type ErrMsg struct{ Err error }

// --- Modal/Form Messages ---
// ShowPinModalMsg is sent *from* the agent *to* the TUI
type ShowPinModalMsg struct {
	DevicePath dbus.ObjectPath
}

// ShowConfirmModalMsg is sent *from* the agent *to* the TUI
type ShowConfirmModalMsg struct {
	DeviceName string
	DevicePath dbus.ObjectPath
	Passkey    uint32
}

// SubmitPinMsg is sent *from* the TUI (modal) *to* the Update loop
type SubmitPinMsg struct {
	Pin string
}

// SubmitConfirmMsg is sent *from* the TUI (modal) *to* the Update loop
type SubmitConfirmMsg struct {
	Confirmed bool
}

// --- Data Models ---

// Adapter represents a Bluetooth controller on your computer (e.t.g., hci0).
type Adapter struct {
	Path     			dbus.ObjectPath
	Name     			string // e.g., "My-Laptop (hci0)"
	Address  			string // The adapter's MAC address
	Powered  			bool
	Scanning 			bool // If we are currently discovering
	Modalias 			string // e.g., "usb:v1D6Bp0246d0532"
	Discoverable 	bool
}

// Device represents a remote Bluetooth device (e.g., headset, mouse).
// This single struct replaces *both* KnownNetwork and ScannedNetwork.
type Device struct {
	Path      	dbus.ObjectPath
	Name      	string // e.g., "My Bluetooth Headset"
	Address   	string // The device's MAC address
	Icon      	string // e.g., "audio-headset", "input-mouse"
	AddressType string // e.g., "BR/EDR", "LE"
	Paired    	bool
	Trusted   	bool
	Connected 	bool
	Battery   	int8 // Battery percentage (0-100). -1 if not available.
	
	Connectable bool  // Is the device connectable?
	RSSI				int16 // Signal strength in dBm
}