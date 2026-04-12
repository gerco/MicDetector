# MicDetector

A lightweight macOS daemon that detects whether the microphone and camera are
in use and publishes their status over MQTT. Useful for home automation — e.g.
turning on a "recording" indicator light via Home Assistant.

## How it works

MicDetector polls macOS CoreAudio and CoreMediaIO APIs to check if any process
has an active audio input or video capture session. When the state changes, it
publishes `on` or `off` to MQTT topics. It also reports availability via MQTT
Last Will and Testament, so Home Assistant knows when the machine goes offline.

## Requirements

- macOS (uses Apple frameworks via cgo)
- Xcode Command Line Tools (`xcode-select --install`)
- Go 1.21+
- An MQTT broker
- [just](https://github.com/casey/just) (for install/uninstall)

## Quick start

```bash
# Clone and build
git clone https://git.dries.info/gerco/MicDetector.git
cd MicDetector
go build -o micdetector .

# Create config
mkdir -p ~/Library/Application\ Support/MicDetector
cp config.example.json ~/Library/Application\ Support/MicDetector/config.json
# Edit config.json — set mqtt.broker at minimum
```

## Configuration

Config lives at `~/Library/Application Support/MicDetector/config.json`.

```json
{
  "mqtt": {
    "broker": "tcp://192.168.1.100:1883",
    "username": "",
    "password": "",
    "client_id": "micdetector",
    "topic_prefix": "micdetector"
  },
  "hostname": "",
  "poll_interval": "2s",
  "homeassistant_discovery": false,
  "log_level": "info"
}
```

Only `mqtt.broker` is required. Everything else has sensible defaults.

## MQTT topics

```
micdetector/<serial>/microphone/state   → "on" or "off"
micdetector/<serial>/camera/state       → "on" or "off"
micdetector/<serial>/status             → "online" or "offline"
```

The serial number is the Mac's hardware serial (lowercase), used as a stable
identifier. All messages are retained at QoS 1.

## Home Assistant

Set `"homeassistant_discovery": true` in config. MicDetector will publish MQTT
auto-discovery configs, and the sensors will appear in Home Assistant
automatically. Availability tracking is included — sensors show as
"unavailable" when the machine is offline.

## Installation

Use [just](https://github.com/casey/just) to install as a launchd user agent
(runs when you're logged in, restarts automatically):

```bash
just install
```

This will:
- Build the binary
- Copy it to `~/bin/micdetector`
- Install the config (if it doesn't exist yet)
- Install and load a launchd agent

Other commands:

```bash
just restart    # Rebuild and restart
just uninstall  # Stop and remove (keeps config)
just status     # Check if running
just logs       # Tail the log file
```

## macOS permissions

On first run, macOS will prompt for microphone and camera access. Grant
permission in System Settings > Privacy & Security. This is required even
though MicDetector only checks device state — it does not capture audio or
video.

## License

MIT
