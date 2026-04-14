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
- An MQTT broker

## Installation

```bash
brew install gerco/micdetector/micdetector
```

Edit the config — set `mqtt.broker` at minimum:

```bash
edit ~/Library/Application\ Support/MicDetector/config.json
```

Start as a background service:

```bash
brew services start micdetector
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

## Viewing logs

MicDetector logs to Apple's unified logging system:

```bash
log stream --predicate 'subsystem == "com.micdetector"' --style compact
```

## Development

Requires Go 1.21+ and Xcode Command Line Tools (`xcode-select --install`).

A [justfile](https://github.com/casey/just) is included for development:

```bash
just build      # Build the binary
just install    # Build, install, and start as launchd agent
just restart    # Rebuild and restart
just uninstall  # Stop and remove (keeps config)
just status     # Check if running
just logs       # View recent logs
just logs-stream # Stream logs in real time
```

## macOS permissions

On first run, macOS will prompt for microphone and camera access. Grant
permission in System Settings > Privacy & Security. This is required even
though MicDetector only checks device state — it does not capture audio or
video.

## License

MIT
