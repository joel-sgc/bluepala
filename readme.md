# bluepala (Impala Go Edition)

A lightweight (hopefully), terminal-friendly **NetworkManager + wpa_supplicant** wrapper written in **Go**.
It’s a clone of **Impala** because Impala's UI made a white tear roll down my leg.

---

## 🚀 Timeline

- ✅ Lists available **bluetooth adapters**
- ✅ Displays **paired** and **scanned** devices
- ❌ Pairing and unpairing devices
- ❌ Connecting and disconnecting to and from devices
- ❌ Showing devices details (Name, MAC address, Path, RSSID, Battery, Type)
- ❌ Handle LE and regular devices

---

## ⚠️ What’s Missing / TODO

- VPN connections manager (halfway down)
- Probably some bugs

It’s functional enough for me right now, but PRs are welcome if you want to polish it up.

---

## 🧩 Implementation Notes

The DBus code was **vibe-coded**.
Yes, really. It works, I don’t care, and it’s not that deep.
If that sets you off, feel free to fork it, rewrite it, etc... Do whatever, idc.

---

## 🛠️ Build & Run

\# Clone and build

\```bash
git clone https://github.com/joel-sgc/bluepala.git
cd bluepala
go build
./bluepala
\```

Then, edit your omarchy-launch-wifi script to:

```bash
#!/bin/bash
exec setsid uwsm app -- "$TERMINAL" --class=Impala -e ~/bluepala/bluepala "$@"
```

You’ll need:

- Go 1.25.1+ (New to go, but this is the version I used so hopefully it works for you too)
- NetworkManager running
- dbus available

---

## 🧾 License

**Do What the Fuck You Want To Public License (WTFPL License)**

---

## ❤️ Closing Thoughts

I built this for myself because I wanted something that just works — and it does.
If you like it, awesome. If not, feel free to improve it or ignore it entirely.
