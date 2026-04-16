# MicDetector

A lightweight cross-platform daemon that detects whether the microphone and camera are
in use and publishes their status over MQTT. Useful for home automation — e.g.
turning on a "recording" indicator light via Home Assistant.

## How it works

MicDetector polls system APIs (CoreAudio/CoreMediaIO on macOS, WASAPI/DirectShow on Windows)
to check if any process has an active audio input or video capture session. When the state
changes, it publishes `on` or `off` to MQTT topics. It also reports availability via MQTT
Last Will and Testament, so Home Assistant knows when the machine goes offline.

## Requirements

- macOS or Windows
- An MQTT broker

### Platform-Specific Requirements

**macOS**: Xcode Command Line Tools (`xcode-select --install`)

**Windows**: MinGW-w64 with GCC (required for CGO - MSVC is not supported by Go)

## Installation

### macOS

```bash
brew install gerco/tap/micdetector
brew services start micdetector
```

On first start, MicDetector creates a default config file and stops.
Edit it to set your MQTT broker address, then restart the service:

```bash
$EDITOR ~/Library/Application\ Support/MicDetector/config.json
brew services restart micdetector
```

### Windows

Download the latest Windows release from [GitHub Releases](https://github.com/gerco/MicDetector/releases),
extract to a directory in your PATH, and run:

```powershell
micdetector.exe
```

On first start, MicDetector will create a default config file at:
`%APPDATA%\MicDetector\config.json`

Edit this file to set your MQTT broker address, then restart the application.

## Configuration

Config lives at:
- **macOS**: `~/Library/Application Support/MicDetector/config.json`
- **Windows**: `%APPDATA%\MicDetector\config.json`

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

The serial number is the machine's hardware serial on macOS, or a stable
machine identifier on Windows (lowercase). All messages are retained at QoS 1.

## Home Assistant

Set `"homeassistant_discovery": true` in config. MicDetector will publish MQTT
auto-discovery configs, and the sensors will appear in Home Assistant
automatically. Availability tracking is included — sensors show as
"unavailable" when the machine is offline.

## Viewing logs

### macOS

MicDetector logs to Apple's unified logging system:

```bash
log stream --predicate 'subsystem == "com.micdetector"' --style compact
```

### Windows

MicDetector logs to standard output (console). Run from command prompt or
PowerShell to view logs:

```powershell
micdetector.exe
```

## Development

Requires Go 1.21+ and platform-specific build tools.

### macOS

Install Xcode Command Line Tools:
```bash
xcode-select --install
```

### Windows

Install MinGW-w64 with WinLibs (provides GCC for CGO):

```powershell
winget install BrechtSanders.WinLibs.POSIX.UCRT
```

Or download from [WinLibs releases](https://github.com/brechtsanders/winlibs_mingw/releases).

**Note:** MSVC is not supported by Go's CGO. MinGW is required.

### Building

```bash
go build .
```

A [justfile](https://github.com/casey/just) is included for macOS development:

```bash
just build      # Build the binary
just install    # Build, install, and start as launchd agent
just restart    # Rebuild and restart
just uninstall  # Stop and remove (keeps config)
just status     # Check if running
just logs       # View recent logs
just logs-stream # Stream logs in real time
```

## Platform Permissions

### macOS

On first run, macOS will prompt for microphone and camera access. Grant
permission in System Settings > Privacy & Security. This is required even
though MicDetector only checks device state — it does not capture audio or
video.

### Windows

On first run, Windows may prompt for microphone access. Grant permission in
Windows Settings > Privacy > Microphone. Note: Windows camera "in-use" detection
currently detects camera presence rather than active usage (full usage detection
requires additional implementation).

## License

MIT
