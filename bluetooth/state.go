package bluetooth

import (
	"bluepala/common"
	"fmt"

	"github.com/godbus/dbus/v5"
)

// BlueZ D-Bus constants
const (
	BluezDest       = "org.bluez"
	BluezPath       = "/"
	ObjectManagerIF = "org.freedesktop.DBus.ObjectManager"
	AdapterIF       = "org.bluez.Adapter1"
	DeviceIF        = "org.bluez.Device1"
	BatteryIF       = "org.bluez.Battery1" // For battery level
	PropsIF         = "org.freedesktop.DBus.Properties"
)

var BluetoothTypes = map[string]string{
	"audio_headset":  "Headset",
	"audio_speaker":  "Speaker",
	"input_mouse":    "Mouse  ",
	"input_keyboard": "Keybd  ",
	"input_gamepad":  "Gamepd ",
	"phone":          "Phone  ",
	"computer":       "PC     ",
	"watch":          "Watch  ",
	"tablet":         "Tablet ",
	"printer":        "Print  ",
	"modem":          "Modem  ",
	"display":        "Screen ",
	"unknown":        "Other  ",
}

// ManagedObjects is the complex type returned by GetManagedObjects
// map[ObjectPath] -> map[InterfaceName] -> map[PropertyName] -> Variant
type ManagedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

// GetInitialState fetches the complete BlueZ state in one call.
// This is the synchronous data-gathering function, like Netpala's GetDevicesData.
func GetInitialState(conn *dbus.Conn) ([]common.Adapter, []common.Device, error) {
	var objects ManagedObjects
	obj := conn.Object(BluezDest, BluezPath)

	err := obj.Call(ObjectManagerIF+".GetManagedObjects", 0).Store(&objects)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call GetManagedObjects: %w", err)
	}

	// Now, parse this map into our structs
	var adapters []common.Adapter
	var devices []common.Device

	for path, interfaces := range objects {
		// Is it an Adapter?
		if props, ok := interfaces[AdapterIF]; ok {
			adapter := ParseAdapter(conn, path, props)
			adapters = append(adapters, adapter)
		}

		// Is it a Device?
		if props, ok := interfaces[DeviceIF]; ok {
			// Check for battery info at the same path
			batteryProps := interfaces[BatteryIF] // Will be nil if not present
			device := ParseDevice(path, props, batteryProps)
			devices = append(devices, device)
		}
	}

	return adapters, devices, nil
}

// --- Helper Functions for Parsing ---

// parseAdapter converts a property map to our common.Adapter struct
func ParseAdapter(conn *dbus.Conn, path dbus.ObjectPath, props map[string]dbus.Variant) common.Adapter {
	adapter := common.Adapter{Path: path}

	// Safely read each property
	if name, ok := props["Name"]; ok {
		adapter.Name, _ = name.Value().(string)
	}
	if addr, ok := props["Address"]; ok {
		adapter.Address, _ = addr.Value().(string)
	}
	if powered, ok := props["Powered"]; ok {
		adapter.Powered, _ = powered.Value().(bool)
	}
	// Note: BlueZ uses "Discovering" for scanning
	if scanning, ok := props["Discovering"]; ok {
		adapter.Scanning, _ = scanning.Value().(bool)
	}

	// Fancy name extraction from Modalias
	if modalias, ok := props["Modalias"]; ok {
		adapter.Modalias, _ = modalias.Value().(string)
	}

	if status, err := IsAdapterDiscoverable(conn, path); err == nil {
		adapter.Discoverable = status
	}

	return adapter
}

// ParseDevice converts property maps to our common.Device struct
func ParseDevice(path dbus.ObjectPath, props, batteryProps map[string]dbus.Variant) common.Device {
	// Default battery to -1 (unavailable)
	device := common.Device{Path: path, Battery: -1, RSSI: 0}

	// Keep alias after
	if name, ok := props["Name"]; ok {
		device.Name = common.SanitizeEmojis(name.Value().(string), "[?]")
	}
	if alias, ok := props["Alias"]; ok {
		device.Name = common.SanitizeEmojis(alias.Value().(string), "[?]")
	}

	if addr, ok := props["Address"]; ok {
		device.Address, _ = addr.Value().(string)
	}
	if addrType, ok := props["AddressType"]; ok {
		device.AddressType, _ = addrType.Value().(string)
	}
	if icon, ok := props["Icon"]; ok {
		device.Icon = NormalizeIcon(icon.Value().(string))
	}
	if paired, ok := props["Paired"]; ok {
		device.Paired, _ = paired.Value().(bool)
	}
	if trusted, ok := props["Trusted"]; ok {
		device.Trusted, _ = trusted.Value().(bool)
	}
	if connected, ok := props["Connected"]; ok {
		device.Connected, _ = connected.Value().(bool)
	}
	if connectable, ok := props["Connectable"]; ok {
		device.Connectable, _ = connectable.Value().(bool)
	}
  if rssi, ok := props["RSSI"]; ok {
		device.RSSI = rssi.Value().(int16)
	}

	// If battery properties exist, try to parse them
	if batteryProps != nil {
		if percent, ok := batteryProps["Percentage"]; ok {
			// Battery is a 'byte' (uint8)
			if val, ok := percent.Value().(byte); ok {
				device.Battery = int8(val)
			}
		}
	}

	return device
}

// IsAdapterDiscoverable checks if a specific Bluetooth adapter is discoverable.
// 'adapterPath' is the D-Bus path, e.g., "/org/bluez/hci0"
func IsAdapterDiscoverable(conn *dbus.Conn, adapterPath dbus.ObjectPath) (bool, error) {
	// 1. Get the D-Bus object for the adapter
	obj := conn.Object(BluezDest, adapterPath)

	// 2. Call the "Get" method on the "Properties" interface
	// We ask for the "Discoverable" property on the "org.bluez.Adapter1" interface
	variant, err := obj.GetProperty(AdapterIF + ".Discoverable")
	if err != nil {
		return false, fmt.Errorf("failed to get Discoverable property: %w", err)
	}

	// 3. Assert the type of the returned variant
	// The 'Discoverable' property is a boolean
	discoverable, ok := variant.Value().(bool)
	if !ok {
		return false, fmt.Errorf("discoverable property was not a boolean")
	}

	return discoverable, nil
}

func NormalizeIcon(icon string) string {
    switch icon {
    case "audio-card", "audio-headset", "audio-headphones", "audio-speaker":
        return "Audio"
    case "input-keyboard":
        return "Keyboard"
    case "input-mouse":
        return "Mouse"
    case "input-tablet":
        return "Tablet"
    case "input-gaming":
        return "Controller"
    case "phone":
        return "Phone"
    case "computer", "computer-laptop":
        return "Computer"
    case "camera":
        return "Camera"
    case "printer":
        return "Printer"
    case "network-wireless":
        return "Net Adapter"
    case "other":
        return "Other"
    default:
        return "Unknown"
    }
}
