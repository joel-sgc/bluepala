package main

import (
	"flag"
	"fmt"
	"log"
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

type BluetoothScanner struct {
	conn        *dbus.Conn
	adapterPath dbus.ObjectPath
}

// NewBluetoothScanner creates a new Bluetooth scanner instance
func NewBluetoothScanner() (*BluetoothScanner, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %v", err)
	}

	scanner := &BluetoothScanner{
		conn: conn,
	}

	// Find the default adapter
	adapterPath, err := scanner.findDefaultAdapter()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to find default Bluetooth adapter: %v", err)
	}

	scanner.adapterPath = adapterPath
	fmt.Printf("Using Bluetooth adapter: %s\n", adapterPath)
	return scanner, nil
}

// Close closes the D-Bus connection
func (bs *BluetoothScanner) Close() {
	bs.conn.Close()
}

// findDefaultAdapter finds the first available Bluetooth adapter
func (bs *BluetoothScanner) findDefaultAdapter() (dbus.ObjectPath, error) {
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := bs.conn.Object(bluezBusName, "/").Call(objectManagerInterface+".GetManagedObjects", 0).Store(&objects)
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

// StartScanning starts Bluetooth device discovery
func (bs *BluetoothScanner) StartScanning() error {
	obj := bs.conn.Object(bluezBusName, bs.adapterPath)
	call := obj.Call(bluezAdapterInterface+".StartDiscovery", 0)
	if call.Err != nil {
		return fmt.Errorf("failed to start discovery: %v", call.Err)
	}
	fmt.Println("✅ Bluetooth scanning started")
	return nil
}

// StopScanning stops Bluetooth device discovery
func (bs *BluetoothScanner) StopScanning() error {
	obj := bs.conn.Object(bluezBusName, bs.adapterPath)
	call := obj.Call(bluezAdapterInterface+".StopDiscovery", 0)
	if call.Err != nil {
		return fmt.Errorf("failed to stop discovery: %v", call.Err)
	}
	fmt.Println("❌ Bluetooth scanning stopped")
	return nil
}

// ToggleScanning toggles the scanning state
func (bs *BluetoothScanner) ToggleScanning() error {
	isScanning, err := bs.IsScanning()
	if err != nil {
		return fmt.Errorf("failed to get scanning state: %v", err)
	}

	if isScanning {
		return bs.StopScanning()
	} else {
		return bs.StartScanning()
	}
}

// IsScanning checks if the adapter is currently scanning
func (bs *BluetoothScanner) IsScanning() (bool, error) {
	obj := bs.conn.Object(bluezBusName, bs.adapterPath)
	variant, err := obj.GetProperty(bluezAdapterInterface + ".Discovering")
	if err != nil {
		return false, err
	}

	discovering, ok := variant.Value().(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type for Discovering property")
	}

	return discovering, nil
}

// GetScanningStatus returns the current scanning status as a string
func (bs *BluetoothScanner) GetScanningStatus() (string, error) {
	isScanning, err := bs.IsScanning()
	if err != nil {
		return "", err
	}

	if isScanning {
		return "Scanning is ACTIVE", nil
	}
	return "Scanning is INACTIVE", nil
}

// ScanForDuration starts scanning and runs for the specified duration, watching for devices
func (bs *BluetoothScanner) ScanForDuration(duration time.Duration) error {
	fmt.Printf("Starting scan for %v...\n", duration)
	
	// Start discovery
	if err := bs.StartScanning(); err != nil {
		return err
	}

	// Set up signal matching for new devices
	err := bs.conn.AddMatchSignal(
		dbus.WithMatchInterface(objectManagerInterface),
		dbus.WithMatchMember("InterfacesAdded"),
	)
	if err != nil {
		return fmt.Errorf("failed to add signal match: %v", err)
	}

	signals := make(chan *dbus.Signal, 10)
	bs.conn.Signal(signals)

	// Create timer for the scan duration
	timer := time.NewTimer(duration)
	defer timer.Stop()

	fmt.Println("Listening for Bluetooth devices...")
	fmt.Println("Press Ctrl+C to stop early")

	for {
		select {
		case signal := <-signals:
			if signal.Name == objectManagerInterface+".InterfacesAdded" {
				bs.handleNewDevice(signal)
			}
		case <-timer.C:
			fmt.Println("\nScan duration completed")
			bs.StopScanning()
			return nil
		}
	}
}

// handleNewDevice processes signals for newly discovered devices
func (bs *BluetoothScanner) handleNewDevice(signal *dbus.Signal) {
	if len(signal.Body) >= 2 {
		if path, ok := signal.Body[0].(dbus.ObjectPath); ok {
			if interfaces, ok := signal.Body[1].(map[string]map[string]dbus.Variant); ok {
				for intf, props := range interfaces {
					if intf == bluezDeviceInterface {
						// The extractDeviceInfo function expects a map of interfaces,
						// so we wrap the device properties in a map with the interface name as the key.
						deviceInterfaces := map[string]map[string]dbus.Variant{
							bluezDeviceInterface: props,
						}
						deviceInfo := bs.extractDeviceInfo(deviceInterfaces)
						fmt.Printf("📱 Found device: %s (%s) - %s\n",
							deviceInfo["Name"], deviceInfo["Address"], path)
					}
				}
			}
		}
	}
}

// extractDeviceInfo extracts device information from properties
func (bs *BluetoothScanner) extractDeviceInfo(props map[string]map[string]dbus.Variant) map[string]string {
	info := make(map[string]string)
	
	if nameVar, exists := props[bluezDeviceInterface]["Name"]; exists {
		if name, ok := nameVar.Value().(string); ok {
			info["Name"] = name
		}
	}
	
	if addrVar, exists := props[bluezDeviceInterface]["Address"]; exists {
		if addr, ok := addrVar.Value().(string); ok {
			info["Address"] = addr
		}
	}
	
	if info["Name"] == "" {
		info["Name"] = "Unknown"
	}
	
	return info
}

// GetDiscoveredDevices returns a list of currently discovered devices
func (bs *BluetoothScanner) GetDiscoveredDevices() ([]map[string]string, error) {
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := bs.conn.Object(bluezBusName, "/").Call(objectManagerInterface+".GetManagedObjects", 0).Store(&objects)
	if err != nil {
		return nil, err
	}

	var devices []map[string]string
	for path, interfaces := range objects {
		if _, exists := interfaces[bluezDeviceInterface]; exists {
			deviceInfo := bs.extractDeviceInfo(interfaces)
			deviceInfo["Path"] = string(path)
			devices = append(devices, deviceInfo)
		}
	}

	return devices, nil
}

// PrintDiscoveredDevices prints all currently discovered devices
func (bs *BluetoothScanner) PrintDiscoveredDevices() error {
	devices, err := bs.GetDiscoveredDevices()
	if err != nil {
		return err
	}

	fmt.Printf("\n📋 Discovered Devices (%d):\n", len(devices))
	for i, device := range devices {
		fmt.Printf("  %d. %s (%s)\n", i+1, device["Name"], device["Address"])
	}
	return nil
}

func main() {
	// Define command line flags
	start := flag.Bool("start", false, "Start Bluetooth scanning")
	stop := flag.Bool("stop", false, "Stop Bluetooth scanning")
	toggle := flag.Bool("toggle", false, "Toggle Bluetooth scanning")
	status := flag.Bool("status", false, "Check scanning status")
	scan := flag.Duration("scan", 0*time.Second, "Scan for specified duration (e.g., 30s, 1m)")
	list := flag.Bool("list", false, "List discovered devices")
	flag.Parse()

	// Create Bluetooth scanner
	scanner, err := NewBluetoothScanner()
	if err != nil {
		log.Fatal("Error:", err)
	}
	defer scanner.Close()

	// Execute commands based on flags
	switch {
	case *start:
		err = scanner.StartScanning()
	case *stop:
		err = scanner.StopScanning()
	case *toggle:
		err = scanner.ToggleScanning()
	case *status:
		status, err := scanner.GetScanningStatus()
		if err != nil {
			log.Fatal("Error:", err)
		}
		fmt.Println(status)
	case *scan > 0:
		err = scanner.ScanForDuration(*scan)
		if err == nil {
			scanner.PrintDiscoveredDevices()
		}
	case *list:
		err = scanner.PrintDiscoveredDevices()
	default:
		fmt.Println("Bluetooth Scanner Tool")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  ./bluetooth-scanner -start")
		fmt.Println("  ./bluetooth-scanner -scan 30s")
		fmt.Println("  ./bluetooth-scanner -status")
		fmt.Println("  ./bluetooth-scanner -toggle")
	}

	if err != nil {
		log.Fatal("Error:", err)
	}
}