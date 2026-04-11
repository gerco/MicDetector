home_dir := env("HOME")
bin_dir := home_dir / "bin"
binary := bin_dir / "micdetector"
plist_name := "com.micdetector.plist"
plist_dest := home_dir / "Library" / "LaunchAgents" / plist_name
config_dir := home_dir / "Library" / "Application Support" / "MicDetector"

# Build the binary
build:
    go build -o micdetector .

# Install binary, config, and launchd agent
install: build
    mkdir -p "{{bin_dir}}"
    cp micdetector "{{binary}}"
    mkdir -p "{{config_dir}}"
    @if [ ! -f "{{config_dir}}/config.json" ]; then \
        cp config.example.json "{{config_dir}}/config.json"; \
        echo "Installed example config to {{config_dir}}/config.json — edit it with your MQTT broker address"; \
    else \
        echo "Config already exists at {{config_dir}}/config.json, not overwriting"; \
    fi
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

# View logs
logs:
    @tail -50 /tmp/micdetector.log 2>/dev/null || echo "No log file yet"
