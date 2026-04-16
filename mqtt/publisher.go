package mqtt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher handles MQTT connections and publishing device state.
type Publisher struct {
	client       pahomqtt.Client
	topicPrefix  string
	hostname     string
	serialNumber string
	logger       *slog.Logger
}

// Config holds the parameters needed to create a Publisher.
type Config struct {
	Broker       string
	Username     string
	Password     string
	ClientID     string
	TopicPrefix  string
	Hostname     string
	SerialNumber string
}

// NewPublisher creates and connects an MQTT publisher.
func NewPublisher(cfg Config, logger *slog.Logger) (*Publisher, error) {
	p := &Publisher{
		topicPrefix:  cfg.TopicPrefix,
		hostname:     cfg.Hostname,
		serialNumber: cfg.SerialNumber,
		logger:       logger,
	}

	opts := pahomqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
			logger.Warn("MQTT connection lost", "error", err)
		}).
		SetOnConnectHandler(func(_ pahomqtt.Client) {
			logger.Info("MQTT connected", "broker", cfg.Broker)
			p.publishAvailability("online")
		})

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	// LWT: broker publishes "offline" if we disconnect unexpectedly.
	availTopic := p.availabilityTopic()
	opts.SetWill(availTopic, "offline", 1, true)

	p.client = pahomqtt.NewClient(opts)
	token := p.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("MQTT connect timed out")
	}
	if token.Error() != nil {
		return nil, fmt.Errorf("MQTT connect: %w", token.Error())
	}

	return p, nil
}

func (p *Publisher) availabilityTopic() string {
	return fmt.Sprintf("%s/%s/status", p.topicPrefix, p.serialNumber)
}

func (p *Publisher) publishAvailability(payload string) {
	topic := p.availabilityTopic()
	token := p.client.Publish(topic, 1, true, payload)
	if !token.WaitTimeout(5 * time.Second) {
		p.logger.Error("availability publish timed out", "topic", topic)
		return
	}
	if token.Error() != nil {
		p.logger.Error("availability publish failed", "topic", topic, "error", token.Error())
		return
	}
	p.logger.Info("published", "topic", topic, "payload", payload)
}

// Publish sends a device state update over MQTT.
// device is "microphone" or "camera". on indicates whether the device is active.
func (p *Publisher) Publish(device string, on bool) {
	topic := fmt.Sprintf("%s/%s/%s/state", p.topicPrefix, p.serialNumber, device)
	payload := "off"
	if on {
		payload = "on"
	}

	token := p.client.Publish(topic, 1, true, payload)
	if !token.WaitTimeout(5 * time.Second) {
		p.logger.Error("MQTT publish timed out", "topic", topic)
		return
	}
	if token.Error() != nil {
		p.logger.Error("MQTT publish failed", "topic", topic, "error", token.Error())
		return
	}
	p.logger.Info("published", "topic", topic, "payload", payload)
}

// discoveryPayload is the JSON structure for Home Assistant MQTT discovery.
type discoveryPayload struct {
	Name              string    `json:"name"`
	StateTopic        string    `json:"state_topic"`
	PayloadOn         string    `json:"payload_on"`
	PayloadOff        string    `json:"payload_off"`
	DeviceClass       string    `json:"device_class,omitempty"`
	UniqueID          string    `json:"unique_id"`
	ObjectID          string    `json:"object_id"`
	AvailabilityTopic string    `json:"availability_topic,omitempty"`
	Device            *haDevice `json:"device,omitempty"`
}

type haDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer,omitempty"`
}

// PublishHADiscovery publishes Home Assistant MQTT auto-discovery configs
// for both the microphone and camera sensors.
func (p *Publisher) PublishHADiscovery() {
	sensors := []struct {
		device      string
		name        string
		deviceClass string
	}{
		{"microphone", "Microphone", ""},
		{"camera", "Camera", ""},
	}

	deviceInfo := &haDevice{
		Identifiers: []string{fmt.Sprintf("micdetector_%s", p.serialNumber)},
		Name:        fmt.Sprintf("MicDetector (%s)", p.hostname),
	}

	for _, s := range sensors {
		objectID := fmt.Sprintf("micdetector_%s_%s", p.serialNumber, s.device)
		stateTopic := fmt.Sprintf("%s/%s/%s/state", p.topicPrefix, p.serialNumber, s.device)
		discoveryTopic := fmt.Sprintf("homeassistant/binary_sensor/%s/config", objectID)

		payload := discoveryPayload{
			Name:              fmt.Sprintf("%s %s", p.hostname, s.name),
			StateTopic:        stateTopic,
			PayloadOn:         "on",
			PayloadOff:        "off",
			DeviceClass:       s.deviceClass,
			UniqueID:          objectID,
			ObjectID:          objectID,
			AvailabilityTopic: p.availabilityTopic(),
			Device:            deviceInfo,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			p.logger.Error("failed to marshal HA discovery payload", "error", err)
			continue
		}

		token := p.client.Publish(discoveryTopic, 1, true, data)
		if !token.WaitTimeout(5 * time.Second) {
			p.logger.Error("HA discovery publish timed out", "topic", discoveryTopic)
			continue
		}
		if token.Error() != nil {
			p.logger.Error("HA discovery publish failed", "topic", discoveryTopic, "error", token.Error())
			continue
		}
		p.logger.Info("published HA discovery", "topic", discoveryTopic)
	}
}

// Disconnect publishes offline status and cleanly disconnects from the MQTT broker.
func (p *Publisher) Disconnect() {
	p.publishAvailability("offline")
	p.client.Disconnect(1000) // wait up to 1s for in-flight messages
	p.logger.Info("MQTT disconnected")
}
