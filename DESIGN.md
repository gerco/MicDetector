# MicDetector - macOS Microphone & Camera Status MQTT Reporter

## Goal

A lightweight daemon that detects whether the Mac's microphone and camera are
currently in use, and publishes their status over MQTT. Intended for home
automation (e.g. turning on a "recording" light via Home Assistant).

## Language: Go

Go with cgo for the macOS-specific parts. The detection code is inlined
Objective-C++ (`.m` files) calling Apple's C-level CoreAudio and CoreMediaIO
frameworks directly — no external dependency needed for this, as the code is
~100 lines total. The only external dependency is the Eclipse Paho MQTT client.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  MicDetector                     │
│                                                  │
│  ┌──────────────┐   ┌──────────────┐            │
│  │  CoreAudio    │   │ CoreMediaIO  │            │
│  │  (mic state)  │   │ (cam state)  │            │
│  └──────┬───────┘   └──────┬───────┘            │
│         │                  │                     │
│         ▼                  ▼                     │
│  ┌─────────────────────────────────┐            │
│  │        Polling Loop             │            │
│  │   (configurable interval)       │            │
│  └──────────────┬──────────────────┘            │
│                 │                                │
│                 ▼                                │
│  ┌─────────────────────────────────┐            │
│  │     State Change Detector       │            │
│  │  (only publish on transitions)  │            │
│  └──────────────┬──────────────────┘            │
│                 │                                │
│                 ▼                                │
│  ┌─────────────────────────────────┐            │
│  │        MQTT Publisher           │            │
│  │    (Eclipse Paho v3.1.1)        │            │
│  └─────────────────────────────────┘            │
└─────────────────────────────────────────────────┘
```

## Detection Mechanism

### Microphone

Uses **CoreAudio Hardware Abstraction Layer** (`CoreAudio/AudioHardware.h`).
Queries `kAudioDevicePropertyDeviceIsRunningSomewhere` on input devices.
This property returns `1` when any process has an active audio input tap on the
device.

### Camera

Uses **CoreMediaIO Device Abstraction Layer** (`CoreMediaIO/CMIOHardware.h`).
Queries `kCMIODevicePropertyDeviceIsRunningSomewhere` on camera devices.
Same concept as the audio HAL — returns `1` when any process has an active
video capture session.

Both APIs are C-level and work well through cgo. The detection code is
inlined directly in the project as `.m` (Objective-C) files that compile
with cgo's Objective-C support. Based on the approach proven by
[`antonfisher/go-media-devices-state`](https://github.com/antonfisher/go-media-devices-state).

## Device States

Two devices, each with two states:

| Device     | State  | Meaning                              |
|------------|--------|--------------------------------------|
| microphone | `on`   | At least one process is capturing    |
| microphone | `off`  | No process is capturing              |
| camera     | `on`   | At least one process is capturing    |
| camera     | `off`  | No process is capturing              |

The underlying macOS APIs are binary — a device is either in use or not. There
is no distinction between "idle", "recording", "streaming", etc. at the OS
level. Keeping the states simple and honest (`on`/`off`) avoids false
precision.

## MQTT Topics & Payload

### Topic Structure

```
micdetector/<hostname>/microphone/state
micdetector/<hostname>/camera/state
```

The hostname is auto-detected but can be overridden in config.

### Payload

Plain string: `on` or `off`.

### Publish Behavior

- **Retained messages**: Yes. Ensures any subscriber connecting later gets the
  current state immediately.
- **QoS**: 1 (at least once). Good enough for status reporting without the
  overhead of QoS 2.
- **Publish on change only**: The polling loop tracks previous state and only
  publishes when a transition occurs.
- **Startup publish**: Always publishes current state on startup/reconnect,
  regardless of whether a change occurred.

### Home Assistant Auto-Discovery (optional)

Publishes MQTT discovery config to:

```
homeassistant/binary_sensor/micdetector_<hostname>_microphone/config
homeassistant/binary_sensor/micdetector_<hostname>_camera/config
```

With a JSON payload describing the sensor, so Home Assistant picks it up
automatically. This is opt-in via config.

## Configuration

JSON config file (`~/Library/Application Support/MicDetector/config.json`),
parsed with `encoding/json` from the standard library. CLI flag `-config` to
specify a non-default path.

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

All fields except `mqtt.broker` are optional and have sensible defaults.
`hostname` defaults to `os.Hostname()`. `client_id` defaults to
`micdetector-<hostname>`.

## Dependencies

| Dependency | Purpose |
|---|---|
| [`eclipse/paho.mqtt.golang`](https://github.com/eclipse-paho/paho.mqtt.golang) | MQTT 3.1.1 client |

Everything else uses the Go standard library:
- `encoding/json` for configuration
- `log/slog` for structured logging
- `os`, `os/signal` for lifecycle
- `time`, `flag` for polling and CLI args

Device detection is inlined Objective-C compiled via cgo — no external
library.

## Project Structure

```
MicDetector/
├── DESIGN.md
├── go.mod
├── go.sum
├── main.go                  # entry point, wiring
├── config/
│   └── config.go            # configuration loading (encoding/json)
├── detector/
│   ├── detector.go          # polling loop, state change detection
│   ├── camera_darwin.go     # cgo bridge for camera detection
│   ├── camera_darwin.m      # Objective-C: CoreMediaIO queries
│   ├── microphone_darwin.go # cgo bridge for microphone detection
│   └── microphone_darwin.m  # Objective-C: CoreAudio queries
├── mqtt/
│   └── publisher.go         # MQTT connection and publishing
└── config.example.json
```

## Build & Run

```bash
# Build (requires Xcode Command Line Tools for cgo/Objective-C)
go build -o micdetector .

# Run with default config path
./micdetector

# Run with explicit config
./micdetector -config ~/Library/Application\ Support/MicDetector/config.json
```

### Running as a launchd Service

For persistent operation, install as a macOS launch agent:

```xml
<!-- ~/Library/LaunchAgents/com.micdetector.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.micdetector</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/micdetector</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/micdetector.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/micdetector.err</string>
</dict>
</plist>
```

## macOS Permissions

- **Microphone access**: macOS will prompt for microphone permission on first
  run. The binary (or Terminal.app if running from shell) must be granted
  access in System Settings > Privacy & Security > Microphone.
- **Camera access**: Same — requires camera permission grant.
- These permissions are required even though we only *check* the state, not
  actually capture audio/video. The CoreAudio/CoreMediaIO queries touch the
  devices enough to trigger TCC.

## Resolved Decisions

- **Detection code**: Inlined directly, no external dependency. The Objective-C
  code is small and stable — the underlying Apple APIs haven't changed.
- **Process identification**: Not needed. `on`/`off` is sufficient for the
  home automation use case.
- **MQTT version**: v3.1.1 via Paho. The target broker does not support v5.
- **Configuration**: JSON via `encoding/json`. No YAML, no viper.
- **Logging**: `log/slog` from the standard library. No zerolog.
