package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"micdetector/config"
	"micdetector/detector"
	"micdetector/logging"
	"micdetector/mqtt"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", config.DefaultConfigPath(), "path to config file")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if errors.Is(err, config.ErrNotConfigured) {
		fmt.Fprintf(os.Stderr, "MicDetector is not configured yet.\n\nEdit the config file to set your MQTT broker address:\n  %s\n\nThen restart the application.\n\n", *configPath)
		os.Exit(0)
	}
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up slog with the configured level.
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(logging.NewHandler("com.micdetector", "default", level))

	logger.Info("starting micdetector",
		"hostname", cfg.Hostname,
		"poll_interval", cfg.PollInterval,
		"broker", cfg.MQTT.Broker,
	)

	// Connect to MQTT.
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
		os.Exit(1)
	}

	// Publish Home Assistant discovery configs if enabled.
	if cfg.HomeAssistantDiscovery {
		pub.PublishHADiscovery()
	}

	// Set up the detector with a callback that publishes state changes.
	det := detector.New(cfg.PollDuration, func(device string, on bool) {
		pub.Publish(device, on)
	}, logger)

	// Start polling in a goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		det.Run(ctx)
		close(done)
	}()

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)

	cancel()
	<-done

	pub.Disconnect()
	logger.Info("shutdown complete")
}
