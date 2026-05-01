package detector

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// BinaryChangeFunc is called when a binary entity transitions state.
// The entity name is one of "microphone", "camera", "screen_lock".
type BinaryChangeFunc func(entity string, on bool)

// NumericPublishFunc is called on every poll for a numeric entity (e.g. "idle_seconds").
type NumericPublishFunc func(entity string, value int)

// Config holds the runtime configuration of the detector.
type Config struct {
	Interval       time.Duration
	Microphone     bool
	Camera         bool
	ScreenLock     bool
	IdleSeconds    bool
	OnBinaryChange BinaryChangeFunc
	OnNumericValue NumericPublishFunc
	Logger         *slog.Logger
}

// Detector polls the configured entities and dispatches updates via callbacks.
type Detector struct {
	cfg  Config
	prev map[string]bool
}

// New creates a new Detector.
func New(cfg Config) *Detector {
	return &Detector{cfg: cfg, prev: map[string]bool{}}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
// On the first poll it always invokes OnBinaryChange for each enabled binary entity,
// regardless of prior state.
func (d *Detector) Run(ctx context.Context) {
	d.cfg.Logger.Info("starting detector",
		"poll_interval", d.cfg.Interval,
		"microphone", d.cfg.Microphone,
		"camera", d.cfg.Camera,
		"screen_lock", d.cfg.ScreenLock,
		"idle_seconds", d.cfg.IdleSeconds,
	)

	d.poll(true)

	ticker := time.NewTicker(d.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.cfg.Logger.Info("detector stopped")
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

// floatWithTimeout is the float64 variant of boolWithTimeout.
func floatWithTimeout(fn func() float64, timeout time.Duration) (result float64, ok bool) {
	ch := make(chan float64, 1)
	go func() {
		ch <- fn()
	}()
	select {
	case v := <-ch:
		return v, true
	case <-time.After(timeout):
		return 0, false
	}
}

func (d *Detector) poll(forcePublish bool) {
	type binaryResult struct {
		entity string
		on     bool
		ok     bool
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []binaryResult
		idle    int
		idleOk  bool
	)

	runBinary := func(entity string, fn func() bool) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, ok := boolWithTimeout(fn, pollTimeout)
			mu.Lock()
			results = append(results, binaryResult{entity: entity, on: v, ok: ok})
			mu.Unlock()
			if !ok {
				d.cfg.Logger.Warn("check timed out, skipping", "entity", entity)
			}
		}()
	}

	if d.cfg.Microphone {
		runBinary("microphone", IsMicrophoneOn)
	}
	if d.cfg.Camera {
		runBinary("camera", IsCameraOn)
	}
	if d.cfg.ScreenLock {
		runBinary("screen_lock", IsScreenLockedNow)
	}
	if d.cfg.IdleSeconds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, ok := floatWithTimeout(IdleSecondsNow, pollTimeout)
			if !ok {
				d.cfg.Logger.Warn("check timed out, skipping", "entity", "idle_seconds")
				return
			}
			mu.Lock()
			idle = int(v + 0.5)
			idleOk = true
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, r := range results {
		if !r.ok {
			continue
		}
		if forcePublish || d.prev[r.entity] != r.on {
			d.cfg.Logger.Debug("state", "entity", r.entity, "on", r.on)
			d.cfg.OnBinaryChange(r.entity, r.on)
			d.prev[r.entity] = r.on
		}
	}

	if idleOk {
		d.cfg.Logger.Debug("state", "entity", "idle_seconds", "value", strconv.Itoa(idle))
		d.cfg.OnNumericValue("idle_seconds", idle)
	}
}
