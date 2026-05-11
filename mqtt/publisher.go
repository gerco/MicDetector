package mqtt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
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

// EntityFlags reports which entities are enabled. Only enabled entities
// receive a discovery payload.
type EntityFlags struct {
	Microphone  bool
	Camera      bool
	ScreenLock  bool
	IdleSeconds bool
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

func (p *Publisher) stateTopic(entity string) string {
	return fmt.Sprintf("%s/%s/%s/state", p.topicPrefix, p.serialNumber, entity)
}

// publish is the shared internal publish path. quiet controls whether the
// success log is at debug (true) or info (false) level.
func (p *Publisher) publish(topic, payload string, quiet bool) {
	token := p.client.Publish(topic, 1, true, payload)
	if !token.WaitTimeout(5 * time.Second) {
		p.logger.Error("MQTT publish timed out", "topic", topic)
		return
	}
	if token.Error() != nil {
		p.logger.Error("MQTT publish failed", "topic", topic, "error", token.Error())
		return
	}
	if quiet {
		p.logger.Debug("published", "topic", topic, "payload", payload)
	} else {
		p.logger.Info("published", "topic", topic, "payload", payload)
	}
}

// Publish sends a binary entity state update over MQTT (microphone/camera/screen_lock).
func (p *Publisher) Publish(entity string, on bool) {
	payload := "off"
	if on {
		payload = "on"
	}
	p.publish(p.stateTopic(entity), payload, false)
}

// PublishNumeric sends a numeric entity state update over MQTT (idle_seconds).
// Logged at debug level since this fires on every poll cycle.
func (p *Publisher) PublishNumeric(entity string, value int) {
	p.publish(p.stateTopic(entity), strconv.Itoa(value), true)
}

// discoveryPayload is the JSON structure for Home Assistant MQTT discovery.
// Fields apply to either binary_sensor or sensor; unused ones are omitted.
type discoveryPayload struct {
	Name              string    `json:"name"`
	StateTopic        string    `json:"state_topic"`
	PayloadOn         string    `json:"payload_on,omitempty"`
	PayloadOff        string    `json:"payload_off,omitempty"`
	DeviceClass       string    `json:"device_class,omitempty"`
	UnitOfMeasurement string    `json:"unit_of_measurement,omitempty"`
	StateClass        string    `json:"state_class,omitempty"`
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

type haEntity struct {
	entity      string // topic segment: "microphone", "camera", "screen_lock", "idle_seconds"
	displayName string // friendly name suffix
	component   string // "binary_sensor" or "sensor"
	deviceClass string
	unit        string
	stateClass  string
	enabled     bool
}

// PublishHADiscovery publishes Home Assistant MQTT auto-discovery configs
// for each enabled entity.
func (p *Publisher) PublishHADiscovery(flags EntityFlags) {
	entities := []haEntity{
		{entity: "microphone", displayName: "Microphone", component: "binary_sensor", enabled: flags.Microphone},
		{entity: "camera", displayName: "Camera", component: "binary_sensor", enabled: flags.Camera},
		{entity: "screen_lock", displayName: "Screen Lock", component: "binary_sensor", enabled: flags.ScreenLock},
		{entity: "idle_seconds", displayName: "Idle Seconds", component: "sensor", unit: "s", stateClass: "measurement", deviceClass: "duration", enabled: flags.IdleSeconds},
	}

	deviceInfo := &haDevice{
		Identifiers: []string{fmt.Sprintf("micdetector_%s", p.serialNumber)},
		Name:        fmt.Sprintf("MicDetector (%s)", p.serialNumber),
	}

	for _, e := range entities {
		if !e.enabled {
			continue
		}

		objectID := fmt.Sprintf("micdetector_%s_%s", p.serialNumber, e.entity)
		stateTopic := p.stateTopic(e.entity)
		discoveryTopic := fmt.Sprintf("homeassistant/%s/%s/config", e.component, objectID)

		payload := discoveryPayload{
			Name:              e.displayName,
			StateTopic:        stateTopic,
			DeviceClass:       e.deviceClass,
			UnitOfMeasurement: e.unit,
			StateClass:        e.stateClass,
			UniqueID:          objectID,
			ObjectID:          objectID,
			AvailabilityTopic: p.availabilityTopic(),
			Device:            deviceInfo,
		}
		if e.component == "binary_sensor" {
			payload.PayloadOn = "on"
			payload.PayloadOff = "off"
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
