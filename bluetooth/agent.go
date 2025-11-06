package bluetooth

import (
	"bluepala/common"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
)

// BluepalaAgent implements the org.bluez.Agent1 interface
type BluepalaAgent struct {
	// A channel to send messages *back* to the main BubbleTea loop
	// We will use this in the *next* step to show a modal.
	UpdateChan chan<- tea.Msg
	
	// A channel to receive the PIN *from* the main BubbleTea loop
	// This is how the blocking RequestPinCode will work.
	pinResponseChan chan string
}

// NewAgent creates our agent.
func NewAgent() *BluepalaAgent {
	return &BluepalaAgent{
		pinResponseChan: make(chan string), // Internal channel
	}
}

// SetUpdateChan allows the main model to give the agent a way to send messages.
func (a *BluepalaAgent) SetUpdateChan(ch chan<- tea.Msg) {
	a.UpdateChan = ch
}

// --- D-Bus Agent Methods ---

// RequestPinCode is called by BlueZ when a device needs a PIN.
// This method *must* block until it has a PIN to return.
func (a *BluepalaAgent) RequestPinCode(device dbus.ObjectPath) (string, *dbus.Error) {
	log(fmt.Sprintf("RequestPinCode received for %s", device))

	// 1. Send a message to the TUI to show a modal
	if a.UpdateChan != nil {
		a.UpdateChan <- common.ShowPinModalMsg{DevicePath: device}
	} else {
		log("UpdateChan is nil, can't show modal.")
		return "", dbus.MakeFailedError(fmt.Errorf("agent not ready"))
	}
	
	// 2. Block and wait for the TUI to send the PIN back
	log("...waiting for PIN from TUI...")
	pin := <-a.pinResponseChan
	log(fmt.Sprintf("...PIN received: %s", pin))
	
	// 3. Return the PIN to BlueZ
	return pin, nil
}

// SubmitPin is called *by our TUI* to unblock RequestPinCode
func (a *BluepalaAgent) SubmitPin(pin string) {
	a.pinResponseChan <- pin
}

// RequestConfirmation is called by BlueZ for "Just Works" pairing.
func (a *BluepalaAgent) RequestConfirmation(device dbus.ObjectPath, passkey uint32) *dbus.Error {
	log(fmt.Sprintf("RequestConfirmation received for %s with passkey: %d", device, passkey))

	if a.UpdateChan != nil {
		// --- Get the device name to show in the modal ---
		deviceName := "Unknown Device"
		conn, err := dbus.SystemBus() // Get a temporary connection
		if err == nil {
			defer conn.Close()
			obj := conn.Object(BluezDest, device)
			nameVar, err := obj.GetProperty(DeviceIF + ".Name")
			if err == nil {
				if name, ok := nameVar.Value().(string); ok && name != "" {
					deviceName = name
				}
			}
		}
		// ---

		// Ask the TUI to show a "Yes/No" modal
		a.UpdateChan <- common.ShowConfirmModalMsg{
			DeviceName: deviceName,
			DevicePath: device,
			Passkey:    passkey,
		}
	} else {
		log("UpdateChan is nil, can't show modal.")
		return dbus.MakeFailedError(fmt.Errorf("agent not ready"))
	}

	// We'll handle the response in the next step.
	// For now, we just log and wait.
	pin := <-a.pinResponseChan // Re-using pin chan for "yes"/"no"
	log(fmt.Sprintf("...Confirmation received: %s", pin))

	if pin == "yes" {
		return nil
	}
	return dbus.MakeFailedError(fmt.Errorf("pairing rejected"))
}

// SubmitConfirmation is called by our TUI
func (a *BluepalaAgent) SubmitConfirmation(confirmed bool) {
	if confirmed {
		a.pinResponseChan <- "yes"
	} else {
		a.pinResponseChan <- "no"
	}
}


// Release is called by BlueZ when the agent is no longer needed.
func (a *BluepalaAgent) Release() {
	log("Agent Released")
}

// (Other agent methods we don't need, but must exist for the interface)
func (a *BluepalaAgent) AuthorizeService(device dbus.ObjectPath, uuid string) *dbus.Error {
	log(fmt.Sprintf("AuthorizeService for %s, %s", device, uuid))
	return nil // Automatically authorize
}
func (a *BluepalaAgent) RequestAuthorization(device dbus.ObjectPath) *dbus.Error {
	log(fmt.Sprintf("RequestAuthorization for %s", device))
	return nil // Automatically authorize
}
func (a *BluepalaAgent) Cancel() {
	log("Agent Canceled")
}
func (a *BluepalaAgent) DisplayPasskey(device dbus.ObjectPath, passkey uint32, entered uint8) {
	log(fmt.Sprintf("DisplayPasskey: %s, %d", device, passkey))
}
func (a *BluepalaAgent) DisplayPinCode(device dbus.ObjectPath, pincode string) {
	log(fmt.Sprintf("DisplayPinCode: %s, %s", device, pincode))
}


// log is a helper to write to a debug file
func log(msg string) {
	f, _ := os.OpenFile("/tmp/bluepala_agent.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		f.WriteString(msg + "\n")
	}
}