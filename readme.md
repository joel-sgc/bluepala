# bluepala

Bluepala is a lightweight, terminal-first Bluetooth manager written in Go. It provides a simple TUI for inspecting adapters and devices using BlueZ over D-Bus. The UI and UX are inspired by Impala but implemented in Go with Bubble Tea and Lipgloss.

This repository focuses on being small, fast, and pragmatic — useful for managing adapters, viewing paired and nearby devices, and handling Bluetooth agent interactions (PIN/confirmation) via an integrated D-Bus agent.

## Features

- List available Bluetooth adapters
- Show paired devices
- Show scanned/nearby devices (scan toggle)
- Sort scanned devices by RSSI
- React to BlueZ D-Bus signals (device added/removed, properties changed)
- Built-in BlueZ agent implementation to surface PIN / confirmation modals to the TUI

## Current limitations / TODO

- Pairing/unpairing flows are supported via BlueZ but UI polish may be incomplete
- Connect/disconnect flows rely on BlueZ signals — some edge cases may need refinement
- Better handling for LE vs BR/EDR-specific device details
- PIN pairing modals are not integrated yet
- Renaming is not integrated yet

PRs and contributions are welcome — see the notes below.

## Implementation notes

- Language: Go
- UI: Bubble Tea + Lipgloss
- D-Bus: github.com/godbus/dbus/v5
- The app registers a small BlueZ agent to handle PIN and confirmation requests and forwards those events into the TUI via channels.

The codebase is intentionally pragmatic: focused on working behavior over perfection. If you see improvements, submit a PR.

## Build & Run

Requirements

- Go 1.25+ (the project uses modules)
- BlueZ (the system Bluetooth stack)
- D-Bus/system bus available

Build

```bash
git clone https://github.com/joel-sgc/bluepala.git
cd bluepala
go build
```

Run

Run the built binary in a terminal emulator. The app interacts directly with the system D-Bus and BlueZ, so it should be run in a user session that has access to system D-Bus and appropriate permissions.

```bash
./bluepala
```

Notes

- You may need NetworkManager/BlueZ running for useful output.
- Some distributions require specific policies/permissions for non-root access to BlueZ endpoints. If you get permission errors, try running in a session with the correct D-Bus access or consult your distro docs.

## Development

- Project layout (top-level):
	- `bluetooth/` — BlueZ/agent helper code
	- `dbus/` — D-Bus action and event glue (Bubble Tea commands/subscriptions)
	- `common/` — shared models, utils, types
	- `models/` — UI models and table rendering
	- `main` — `bluepala.go` (entrypoint)

To iterate quickly:

```bash
go run .
```

## Contributing

Small, incremental PRs are easiest. If you want to add features, tests, or UI polish, open an issue or a PR.

## License

This project is released under the Do What The Fuck You Want To Public License (WTFPL). See `LICENSE` for full text.

---

If you want any specific additions to this README (examples, screenshots, keybindings, or a quick demo), tell me what you'd like and I can add it.
