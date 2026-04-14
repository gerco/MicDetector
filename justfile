home_dir := env("HOME")
bin_dir := home_dir / "bin"
binary := bin_dir / "micdetector"
plist_name := "com.micdetector.plist"
plist_dest := home_dir / "Library" / "LaunchAgents" / plist_name
config_dir := home_dir / "Library" / "Application Support" / "MicDetector"

# Build the binary
build:
    go build -o micdetector .

# Install binary and launchd agent
install: build
    mkdir -p "{{bin_dir}}"
    cp micdetector "{{binary}}"
    @sed "s|__BINARY__|{{binary}}|g" com.micdetector.plist > "{{plist_dest}}"
    launchctl load "{{plist_dest}}"
    @echo "MicDetector installed and running"

# Stop and remove the launchd agent, binary, and config
uninstall:
    -launchctl unload "{{plist_dest}}"
    rm -f "{{plist_dest}}"
    rm -f "{{binary}}"
    @echo "MicDetector uninstalled (config left in {{config_dir}})"

# Restart the agent (after a rebuild)
restart: build
    cp micdetector "{{binary}}"
    -launchctl unload "{{plist_dest}}"
    launchctl load "{{plist_dest}}"
    @echo "MicDetector restarted"

# Show agent status
status:
    @launchctl list | grep micdetector || echo "Not running"

# View recent logs
logs:
    log show --predicate 'subsystem == "com.micdetector"' --last 5m --style compact

# Stream logs in real time
logs-stream:
    log stream --predicate 'subsystem == "com.micdetector"' --style compact

# Stream logs including debug level
logs-debug:
    log stream --predicate 'subsystem == "com.micdetector"' --level debug --style compact
