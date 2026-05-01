# MicDetector

A lightweight macOS daemon that detects whether the microphone and camera are
in use and publishes their status over MQTT. Useful for home automation — e.g.
turning on a "recording" indicator light via Home Assistant.

## How it works

MicDetector polls macOS CoreAudio, CoreMediaIO, and CoreGraphics APIs to
check the state of four entities, publishing each over MQTT:

- **microphone** — `on` while any process is capturing audio
- **camera** — `on` while any process is capturing video
- **screen_lock** — `on` while the login session reports the screen as locked
- **idle_seconds** — integer seconds since the last input event

The first three are binary; the last is a numeric sensor useful as a
"working" signal that doesn't depend on sleep/wake.

Availability is reported via MQTT Last Will and Testament, so Home Assistant
knows when the machine goes offline.

## Requirements

- macOS (uses Apple frameworks via cgo)
- An MQTT broker

## Installation

```bash
brew install gerco/tap/micdetector
brew services start micdetector
```

On first start, MicDetector creates a default config file and stops.
Edit it to set your MQTT broker address, then restart:

```bash
$EDITOR ~/Library/Application\ Support/MicDetector/config.json
brew services restart micdetector
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
  "log_level": "info",
  "entities": ["microphone", "camera", "screen_lock", "idle_seconds"]
}
```

Only `mqtt.broker` is required. Everything else has sensible defaults.

You can edit the file directly or use the CLI. Every key is settable:

```bash
micdetector config show                                   # print current config (passwords masked)
micdetector config get mqtt.broker
micdetector config set mqtt.broker tcp://192.168.1.100:1883
micdetector config set homeassistant_discovery true
micdetector config set log_level debug
micdetector config unset hostname                         # revert to default
```

`micdetector config --help` and `micdetector config set --help` list every
settable key with its type and description.

The enabled-entities array has its own commands:

```bash
micdetector entities list
micdetector entities disable screen_lock idle_seconds
micdetector entities enable screen_lock
```

`micdetector --help` lists every entity along with a one-line description.

## MQTT topics

```
micdetector/<serial>/microphone/state    → "on" or "off"
micdetector/<serial>/camera/state        → "on" or "off"
micdetector/<serial>/screen_lock/state   → "on" or "off"
micdetector/<serial>/idle_seconds/state  → integer (seconds)
micdetector/<serial>/status              → "online" or "offline"
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
