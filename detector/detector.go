package detector

import (
	"context"
	"log/slog"
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

func (d *Detector) poll(forcePublish bool) {
	micOn := IsMicrophoneOn()
	camOn := IsCameraOn()

	current := State{
		MicrophoneOn: micOn,
		CameraOn:     camOn,
	}

	if forcePublish || current.MicrophoneOn != d.prev.MicrophoneOn {
		d.logger.Debug("microphone state", "on", micOn)
		d.onChange("microphone", micOn)
	}

	if forcePublish || current.CameraOn != d.prev.CameraOn {
		d.logger.Debug("camera state", "on", camOn)
		d.onChange("camera", camOn)
	}

	d.prev = current
}
