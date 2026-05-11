package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"micdetector/config"
	"micdetector/detector"
	"micdetector/logging"
	"micdetector/mqtt"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var configPath string

	root := &cobra.Command{
		Use:     "micdetector",
		Short:   "Publish macOS microphone, camera, screen-lock and idle state over MQTT",
		Long:    rootLongDescription(),
		Version: version,
		// Default action when no subcommand is given: run the daemon.
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDaemon(configPath)
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&configPath, "config", config.DefaultConfigPath(), "path to config file")

	root.AddCommand(
		newEntitiesCmd(&configPath),
		newConfigCmd(&configPath),
	)

	return root
}

func rootLongDescription() string {
	var b strings.Builder
	b.WriteString("MicDetector polls macOS for active microphone/camera use, screen lock\n")
	b.WriteString("state, and seconds since the last input event, then publishes the\n")
	b.WriteString("results over MQTT for use in home automation.\n\n")
	b.WriteString("Available entities:\n")
	for _, e := range config.Available {
		fmt.Fprintf(&b, "  %-13s  %s\n", e.Name, e.Description)
	}
	return b.String()
}

func runDaemon(configPath string) error {
	cfg, err := config.Load(configPath)
	if errors.Is(err, config.ErrNotConfigured) {
		fmt.Fprintf(os.Stderr, `MicDetector is not configured yet.

Edit the config file to set your MQTT broker address:
  %s

Then restart the service:
  brew services restart micdetector

`, configPath)
		return nil
	}
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return err
	}

	logger := slog.New(logging.NewHandler("com.micdetector", "default", parseLogLevel(cfg.LogLevel)))

	logger.Info("starting micdetector",
		"hostname", cfg.Hostname,
		"poll_interval", cfg.PollInterval,
		"broker", cfg.MQTT.Broker,
	)

	pub, err := mqtt.NewPublisher(mqtt.Config{
		Broker:       cfg.MQTT.Broker,
		Username:     cfg.MQTT.Username,
		Password:     cfg.MQTT.Password,
		ClientID:     cfg.MQTT.ClientID,
		TopicPrefix:  cfg.MQTT.TopicPrefix,
		Hostname:     cfg.Hostname,
		SerialNumber: cfg.SerialNumber,
	}, logger)
	if err != nil {
		logger.Error("failed to connect to MQTT broker", "error", err)
		return err
	}

	flags := mqtt.EntityFlags{
		Microphone:  cfg.Entities.IsEnabled("microphone"),
		Camera:      cfg.Entities.IsEnabled("camera"),
		ScreenLock:  cfg.Entities.IsEnabled("screen_lock"),
		IdleSeconds: cfg.Entities.IsEnabled("idle_seconds"),
	}

	if cfg.HomeAssistantDiscovery {
		pub.PublishHADiscovery(flags)
	}

	det := detector.New(detector.Config{
		Interval:    cfg.PollDuration,
		Microphone:  flags.Microphone,
		Camera:      flags.Camera,
		ScreenLock:  flags.ScreenLock,
		IdleSeconds: flags.IdleSeconds,
		OnBinaryChange: func(entity string, on bool) {
			pub.Publish(entity, on)
		},
		OnNumericValue: func(entity string, value int) {
			pub.PublishNumeric(entity, value)
		},
		Logger: logger,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		det.Run(ctx)
		close(done)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	cancel()
	<-done

	pub.Disconnect()
	logger.Info("shutdown complete")
	return nil
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
