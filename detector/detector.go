package detector

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// State represents the current microphone and camera state.
type State struct {
	MicrophoneOn bool
	CameraOn     bool
}

// StateChangeFunc is called whenever a device state changes.
// It receives the device name ("microphone" or "camera") and the new state (true = on).
type StateChangeFunc func(device string, on bool)

// Detector polls microphone and camera state and invokes a callback on changes.
type Detector struct {
	interval time.Duration
	onChange StateChangeFunc
	logger   *slog.Logger
	prev     State
	started  bool
}

// New creates a new Detector with the given poll interval and change callback.
func New(interval time.Duration, onChange StateChangeFunc, logger *slog.Logger) *Detector {
	return &Detector{
		interval: interval,
		onChange: onChange,
		logger:   logger,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
// On the first poll it always invokes the callback for both devices.
func (d *Detector) Run(ctx context.Context) {
	d.logger.Info("starting detector", "poll_interval", d.interval)

	// Do an immediate poll on startup.
	d.poll(true)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("detector stopped")
			return
		case <-ticker.C:
			d.poll(false)
		}
	}
}

// pollTimeout is how long to wait for a single device check before giving up.
// CoreAudio/CoreMediaIO calls can hang when devices are being added/removed.
const pollTimeout = 5 * time.Second

// boolWithTimeout runs fn in a goroutine and returns its result.
// If fn doesn't return within timeout, it returns (false, false).
// The goroutine is left running — this is acceptable because hung
// CoreAudio/CoreMediaIO calls typically unblock once device topology settles.
func boolWithTimeout(fn func() bool, timeout time.Duration) (result bool, ok bool) {
	ch := make(chan bool, 1)
	go func() {
		ch <- fn()
	}()
	select {
	case v := <-ch:
		return v, true
	case <-time.After(timeout):
		return false, false
	}
}

func (d *Detector) poll(forcePublish bool) {
	// Run both checks concurrently with a timeout each.
	var micOn, camOn bool
	var micOk, camOk bool
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		micOn, micOk = boolWithTimeout(IsMicrophoneOn, pollTimeout)
	}()
	go func() {
		defer wg.Done()
		camOn, camOk = boolWithTimeout(IsCameraOn, pollTimeout)
	}()
	wg.Wait()

	if !micOk {
		d.logger.Warn("microphone check timed out, skipping poll cycle")
	}
	if !camOk {
		d.logger.Warn("camera check timed out, skipping poll cycle")
	}

	current := State{
		MicrophoneOn: micOn,
		CameraOn:     camOn,
	}

	if micOk && (forcePublish || current.MicrophoneOn != d.prev.MicrophoneOn) {
		d.logger.Debug("microphone state", "on", micOn)
		d.onChange("microphone", micOn)
		d.prev.MicrophoneOn = current.MicrophoneOn
	}

	if camOk && (forcePublish || current.CameraOn != d.prev.CameraOn) {
		d.logger.Debug("camera state", "on", camOn)
		d.onChange("camera", camOn)
		d.prev.CameraOn = current.CameraOn
	}
}
