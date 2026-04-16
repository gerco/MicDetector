# Platform detection
windows := if os_family() == "windows" { "true" } else { "" }
macos := if os() == "macos" { "true" } else { "" }

# Set shell for Windows
set windows-shell := ["powershell.exe", "-c"]

# Set home directory based on platform
home_dir := if windows == "true" { env("USERPROFILE") } else { env("HOME") }

# macOS-specific paths
bin_dir := home_dir / "bin"
binary := bin_dir / "micdetector"
plist_dest := home_dir / "Library" / "LaunchAgents" / "com.micdetector.plist"
config_dir := home_dir / "Library" / "Application Support" / "MicDetector"

# Build the binary - works on all platforms
# CGO is required for the platform-specific C code (CoreAudio on macOS, WASAPI on Windows)
[unix]
build:
    export CGO_ENABLED=1
    go build .

[windows]
build:
    set CGO_ENABLED=1
    go build .

# Install binary and launchd agent (macOS only)
[macos]
install:
    mkdir -p "{{bin_dir}}"
    cp micdetector "{{binary}}"
    sed "s|__BINARY__|{{binary}}|g" com.micdetector.plist > "{{plist_dest}}"
    launchctl load "{{plist_dest}}"
    echo "MicDetector installed and running"

# Stop and remove the launchd agent, binary, and config (macOS only)
[macos]
uninstall:
    launchctl unload "{{plist_dest}}" 2>/dev/null || echo "Not running"
    rm -f "{{plist_dest}}"
    rm -f "{{binary}}"
    echo "MicDetector uninstalled (config left in {{config_dir}})"

# Restart the agent (after a rebuild) (macOS only)
[macos]
restart:
    cp micdetector "{{binary}}"
    launchctl unload "{{plist_dest}}" 2>/dev/null || echo "Not running"
    launchctl load "{{plist_dest}}"
    echo "MicDetector restarted"

# Show agent status (macOS only)
[macos]
status:
    launchctl list | grep micdetector || echo "Not running"

# View recent logs (macOS only)
[macos]
logs:
    log show --predicate 'subsystem == "com.micdetector"' --last 5m --style compact

# Stream logs in real time (macOS only)
[macos]
logs-stream:
    log stream --predicate 'subsystem == "com.micdetector"' --style compact

# Stream logs including debug level (macOS only)
[macos]
logs-debug:
    log stream --predicate 'subsystem == "com.micdetector"' --level debug --style compact
