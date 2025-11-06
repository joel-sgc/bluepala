package main

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	bluezBusName           = "org.bluez"
	bluezAdapterInterface  = "org.bluez.Adapter1"
	bluezDeviceInterface   = "org.bluez.Device1"
	objectManagerInterface = "org.freedesktop.DBus.ObjectManager"
	propertiesInterface    = "org.freedesktop.DBus.Properties"
)

func main() {
	fmt.Println("🔍 Bluetooth Diagnostic Tool")
	fmt.Println("==============================")

	// Check system status first
	if err := checkBluetoothStatus(); err != nil {
		log.Fatalf("❌ System check failed: %v", err)
	}

	// Set up D-Bus connection
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatal("❌ Failed to connect to system bus:", err)
	}
	defer conn.Close()

	// Set up signal monitoring
	sig := make(chan *dbus.Signal, 100)
	conn.Signal(sig)

	// Add signal matches
	if err := setupSignalMatches(conn); err != nil {
		log.Printf("⚠️ Warning: some signal matches failed: %v", err)
	}

	fmt.Println("\n🎯 Starting Bluetooth discovery...")
	fmt.Println("Listening for devices for 30 seconds...")
	fmt.Println("Make sure Bluetooth is enabled and discoverable!")
	fmt.Println("================================================")

	// Start discovery on the default adapter
	if err := startDiscovery(conn); err != nil {
		log.Printf("⚠️ Could not start discovery: %v", err)
		log.Println("Continuing anyway - there might be existing devices...")
	}

	// Listen for signals for 30 seconds
	timeout := time.After(30 * time.Second)
	deviceCount := 0

	for {
		select {
		case s := <-sig:
			if s.Name == objectManagerInterface+".InterfacesAdded" {
				if len(s.Body) >= 2 {
					if path, ok := s.Body[0].(dbus.ObjectPath); ok {
						if interfaces, ok := s.Body[1].(map[string]map[string]dbus.Variant); ok {
							if _, exists := interfaces[bluezDeviceInterface]; exists {
								deviceCount++

								if IsUsableDevice(interfaces) {
									fmt.Printf("\n=== DEVICE #%d: %s ===\n", deviceCount, path)
									debugDeviceProperties(interfaces)
									testFilters(interfaces)
								}
							}
						}
					}
				}
			} else if s.Name == propertiesInterface+".PropertiesChanged" {
				// Also log property changes for debugging
				if len(s.Body) >= 2 {
					fmt.Printf("\n🔧 Properties changed on %s\n", s.Path)
				}
			}

		case <-timeout:
			fmt.Println("\n================================================")
			fmt.Printf("⏰ Diagnostic completed. Found %d devices.\n", deviceCount)
			
			if deviceCount == 0 {
				fmt.Println("❌ No devices found. Possible issues:")
				fmt.Println("   - Bluetooth adapter not powered on")
				fmt.Println("   - No discoverable devices in range")
				fmt.Println("   - Discovery not started properly")
				fmt.Println("   - Permission issues")
			} else {
				fmt.Println("✅ Devices were detected but may be filtered out")
			}
			
			// Stop discovery before exiting
			stopDiscovery(conn)
			return
		}
	}
}

func IsUsableDevice(interfaces map[string]map[string]dbus.Variant) bool {
	props, exists := interfaces["org.bluez.Device1"]
	if !exists {
		return false
	}

	// Check name (not MAC address)
	if nameVar, exists := props["Name"]; exists {
		if name := nameVar.Value().(string); name != "" {
			// Check if name is NOT a MAC address
			macPattern := `^[0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}$`
			if matched, _ := regexp.MatchString(macPattern, name); !matched {
				return true
			}
		}
	}

	// Check appearance
	if appearanceVar, exists := props["Appearance"]; exists {
		if appearance := appearanceVar.Value().(uint16); appearance != 0 {
			wantedAppearances := map[uint16]bool{
				0x0040: true, // Computer
				0x0140: true, // Phone
				0x0440: true, // Headphones
				0x0441: true, // Headset
				0x0408: true, // Car
				0x0540: true, // Clock
				0x04C0: true, // Wearable
			}
			if wantedAppearances[appearance] {
				return true
			}
		}
	}

	// Check UUIDs
	if uuidsVar, exists := props["UUIDs"]; exists {
		if uuids := uuidsVar.Value().([]string); len(uuids) > 0 {
			usefulServices := map[string]bool{
				"0000110a-0000-1000-8000-00805f9b34fb": true, // A2DP Source
				"0000110b-0000-1000-8000-00805f9b34fb": true, // A2DP Sink  
				"00001108-0000-1000-8000-00805f9b34fb": true, // HSP
				"0000111e-0000-1000-8000-00805f9b34fb": true, // HFP
				"00001112-0000-1000-8000-00805f9b34fb": true, // HID
				"00001124-0000-1000-8000-00805f9b34fb": true, // AVRCP
				"0000180f-0000-1000-8000-00805f9b34fb": true, // Battery
				"0000180a-0000-1000-8000-00805f9b34fb": true, // Device Information
			}
			for _, uuid := range uuids {
				if usefulServices[uuid] {
					return true
				}
			}
		}
	}

	return false
}

func checkBluetoothStatus() error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("D-Bus not available: %v", err)
	}

	// Check if BlueZ is running
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err = conn.Object(bluezBusName, "/").Call(objectManagerInterface+".GetManagedObjects", 0).Store(&objects)
	if err != nil {
		return fmt.Errorf("BlueZ not running or accessible: %v", err)
	}

	// Find adapters
	var adapters []string
	var poweredAdapters []string
	
	for path, interfaces := range objects {
		if adapterProps, exists := interfaces[bluezAdapterInterface]; exists {
			adapterPath := string(path)
			adapters = append(adapters, adapterPath)
			
			// Check if adapter is powered on
			if poweredVar, exists := adapterProps["Powered"]; exists {
				if powered, ok := poweredVar.Value().(bool); ok && powered {
					poweredAdapters = append(poweredAdapters, adapterPath)
				}
			}
		}
	}

	if len(adapters) == 0 {
		return fmt.Errorf("no Bluetooth adapters found")
	}

	fmt.Printf("✅ Found %d adapter(s):\n", len(adapters))
	for i, adapter := range adapters {
		fmt.Printf("   %d. %s\n", i+1, adapter)
	}

	if len(poweredAdapters) == 0 {
		return fmt.Errorf("no powered Bluetooth adapters - please turn on Bluetooth")
	}

	fmt.Printf("✅ %d adapter(s) powered on\n", len(poweredAdapters))
	return nil
}

func setupSignalMatches(conn *dbus.Conn) error {
	matches := []struct {
		iface  string
		member string
	}{
		{objectManagerInterface, "InterfacesAdded"},
		{objectManagerInterface, "InterfacesRemoved"},
		{propertiesInterface, "PropertiesChanged"},
	}

	for _, match := range matches {
		err := conn.AddMatchSignal(
			dbus.WithMatchInterface(match.iface),
			dbus.WithMatchMember(match.member),
		)
		if err != nil {
			return fmt.Errorf("failed to add match for %s.%s: %v", match.iface, match.member, err)
		}
	}
	return nil
}

func findDefaultAdapter(conn *dbus.Conn) (dbus.ObjectPath, error) {
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := conn.Object(bluezBusName, "/").Call(objectManagerInterface+".GetManagedObjects", 0).Store(&objects)
	if err != nil {
		return "", err
	}

	for path, interfaces := range objects {
		if _, exists := interfaces[bluezAdapterInterface]; exists {
			return path, nil
		}
	}

	return "", fmt.Errorf("no Bluetooth adapter found")
}

func startDiscovery(conn *dbus.Conn) error {
	adapterPath, err := findDefaultAdapter(conn)
	if err != nil {
		return err
	}

	obj := conn.Object(bluezBusName, adapterPath)
	call := obj.Call(bluezAdapterInterface+".StartDiscovery", 0)
	if call.Err != nil {
		return call.Err
	}

	fmt.Printf("✅ Started discovery on adapter: %s\n", adapterPath)
	return nil
}

func stopDiscovery(conn *dbus.Conn) {
	adapterPath, err := findDefaultAdapter(conn)
	if err != nil {
		return
	}

	obj := conn.Object(bluezBusName, adapterPath)
	obj.Call(bluezAdapterInterface+".StopDiscovery", 0)
	fmt.Printf("✅ Stopped discovery on adapter: %s\n", adapterPath)
}

func debugDeviceProperties(interfaces map[string]map[string]dbus.Variant) {
	props, exists := interfaces[bluezDeviceInterface]
	if !exists {
		fmt.Println("❌ No Device1 interface found")
		return
	}

	fmt.Println("Raw properties dump:")
	for key, value := range props {
		fmt.Printf("   %s: %v (type: %T)\n", key, value.Value(), value.Value())
	}
}

func testFilters(interfaces map[string]map[string]dbus.Variant) {
	props, exists := interfaces[bluezDeviceInterface]
	if !exists {
		return
	}

	fmt.Println("Filter tests:")
	
	// Test name filter
	name := ""
	if nameVar, exists := props["Name"]; exists {
		name = nameVar.Value().(string)
		fmt.Printf("   📝 Name: '%s' (is MAC: %v)\n", name, isMACAddressName(name))
	} else {
		fmt.Println("   📝 Name: <missing>")
	}

	// Test appearance filter
	if appearanceVar, exists := props["Appearance"]; exists {
		appearance := appearanceVar.Value().(uint16)
		fmt.Printf("   🎯 Appearance: 0x%04x (%d) - wanted: %v\n", 
			appearance, appearance, isWantedAppearance(appearance))
	} else {
		fmt.Println("   🎯 Appearance: <missing>")
	}

	// Test UUIDs filter
	if uuidsVar, exists := props["UUIDs"]; exists {
		uuids := uuidsVar.Value().([]string)
		fmt.Printf("   🔧 UUIDs: %v - useful: %v\n", uuids, hasUsefulServices(uuids))
	} else {
		fmt.Println("   🔧 UUIDs: <missing>")
	}

	// Test RSSI
	if rssiVar, exists := props["RSSI"]; exists {
		rssi := rssiVar.Value().(int16)
		fmt.Printf("   📶 RSSI: %d\n", rssi)
	} else {
		fmt.Println("   📶 RSSI: <missing>")
	}

	// Test if it would pass our filters
	fmt.Printf("   ✅ Would pass filter: %v\n", wouldPassFilter(interfaces))
}

func isMACAddressName(name string) bool {
	if name == "" {
		return false
	}
	// MAC address pattern: XX-XX-XX-XX-XX-XX or XX:XX:XX:XX:XX:XX
	macPattern := `^[0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}$`
	matched, _ := regexp.MatchString(macPattern, name)
	return matched
}

func isWantedAppearance(appearance uint16) bool {
	wantedAppearances := map[uint16]string{
		0x0040: "Computer",
		0x0140: "Phone", 
		0x0440: "Headphones",
		0x0441: "Headset",
		0x0408: "Car",
		0x0540: "Clock",
		0x04C0: "Wearable",
	}
	_, found := wantedAppearances[appearance]
	return found
}

func hasUsefulServices(uuids []string) bool {
	usefulServices := []string{
		"0000110a-0000-1000-8000-00805f9b34fb", // A2DP Source
		"0000110b-0000-1000-8000-00805f9b34fb", // A2DP Sink  
		"00001108-0000-1000-8000-00805f9b34fb", // HSP
		"0000111e-0000-1000-8000-00805f9b34fb", // HFP
		"00001112-0000-1000-8000-00805f9b34fb", // HID
		"00001124-0000-1000-8000-00805f9b34fb", // AVRCP
		"0000180f-0000-1000-8000-00805f9b34fb", // Battery
		"0000180a-0000-1000-8000-00805f9b34fb", // Device Information
	}
	
	for _, uuid := range uuids {
		for _, useful := range usefulServices {
			if uuid == useful {
				return true
			}
		}
	}
	return false
}

func wouldPassFilter(interfaces map[string]map[string]dbus.Variant) bool {
	props, exists := interfaces[bluezDeviceInterface]
	if !exists {
		return false
	}

	// Check if it has a proper name (not MAC address)
	if nameVar, exists := props["Name"]; exists {
		name := nameVar.Value().(string)
		if name != "" && !isMACAddressName(name) {
			return true
		}
	}

	// Check if it has a wanted appearance
	if appearanceVar, exists := props["Appearance"]; exists {
		appearance := appearanceVar.Value().(uint16)
		if isWantedAppearance(appearance) {
			return true
		}
	}

	// Check if it has useful services
	if uuidsVar, exists := props["UUIDs"]; exists {
		uuids := uuidsVar.Value().([]string)
		if hasUsefulServices(uuids) {
			return true
		}
	}

	return false
}