package services

import (
	"connect4/internal/config"
	"connect4/internal/models"
	"connect4/pkg/logger"
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
	"go.uber.org/zap"
)

type KafkaProducer struct {
	writer *kafka.Writer
	config *config.Config
}

func NewKafkaProducer(cfg *config.Config) (*KafkaProducer, error) {
	// Configure SASL/SCRAM authentication for Redpanda
	mechanism, err := scram.Mechanism(scram.SHA256, cfg.Kafka.Username, cfg.Kafka.Password)
	if err != nil {
		return nil, err
	}

	// Create dialer with TLS and SASL
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Topic:        cfg.Kafka.TopicEvents,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Compression:  kafka.Snappy,
		Transport: &kafka.Transport{
			SASL: mechanism,
			TLS:  &tls.Config{},
		},
	}

	kp := &KafkaProducer{
		writer: writer,
		config: cfg,
	}

	logger.Log.Info("Kafka producer initialized",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("topic", cfg.Kafka.TopicEvents),
	)

	return kp, nil
}

func (kp *KafkaProducer) PublishGameStarted(event models.GameStartedEvent) error {
	return kp.publish(event)
}

func (kp *KafkaProducer) PublishMoveMade(event models.MoveMadeEvent) error {
	return kp.publish(event)
}

func (kp *KafkaProducer) PublishGameCompleted(event models.GameCompletedEvent) error {
	return kp.publish(event)
}

func (kp *KafkaProducer) publish(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		logger.Log.Error("Failed to marshal event", zap.Error(err))
		return err
	}

	msg := kafka.Message{
		Key:   []byte(time.Now().Format(time.RFC3339)),
		Value: data,
		Time:  time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = kp.writer.WriteMessages(ctx, msg)
	if err != nil {
		logger.Log.Error("Kafka write failed", zap.Error(err))
		return err
	}

	logger.Log.Debug("Event published to Kafka", zap.Int("size", len(data)))
	return nil
}

func (kp *KafkaProducer) Close() error {
	if kp.writer != nil {
		return kp.writer.Close()
	}
	return nil
}

// Kafka Consumer
type KafkaConsumer struct {
	reader    *kafka.Reader
	analytics *AnalyticsService
}

func NewKafkaConsumer(cfg *config.Config, analytics *AnalyticsService) *KafkaConsumer {
	// Configure SASL for consumer
	mechanism, err := scram.Mechanism(scram.SHA256, cfg.Kafka.Username, cfg.Kafka.Password)
	if err != nil {
		logger.Log.Error("Failed to create SASL mechanism", zap.Error(err))
		return nil
	}

	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		Topic:          cfg.Kafka.TopicEvents,
		GroupID:        "connect4-analytics-consumer",
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		Dialer:         dialer,
	})

	logger.Log.Info("Kafka consumer initialized",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("topic", cfg.Kafka.TopicEvents),
	)

	return &KafkaConsumer{
		reader:    reader,
		analytics: analytics,
	}
}

func (kc *KafkaConsumer) Start(ctx context.Context) {
	logger.Log.Info("Starting Kafka consumer...")

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Kafka consumer stopped")
			return
		default:
			msg, err := kc.reader.ReadMessage(ctx)
			if err != nil {
				logger.Log.Error("Kafka read error", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}

			kc.processMessage(msg)
		}
	}
}

func (kc *KafkaConsumer) processMessage(msg kafka.Message) {
	var baseEvent struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(msg.Value, &baseEvent); err != nil {
		logger.Log.Error("Failed to unmarshal event", zap.Error(err))
		return
	}

	logger.Log.Info("📊 Processing Kafka event",
		zap.String("type", baseEvent.Type),
		zap.Int64("offset", msg.Offset),
	)

	switch models.KafkaEventType(baseEvent.Type) {
	case models.EventGameStarted:
		var event models.GameStartedEvent
		if err := json.Unmarshal(msg.Value, &event); err == nil {
			kc.analytics.ProcessGameStarted(event)
		}
	case models.EventMoveMade:
		var event models.MoveMadeEvent
		if err := json.Unmarshal(msg.Value, &event); err == nil {
			kc.analytics.ProcessMoveMade(event)
		}
	case models.EventGameCompleted:
		var event models.GameCompletedEvent
		if err := json.Unmarshal(msg.Value, &event); err == nil {
			kc.analytics.ProcessGameCompleted(event)
		}
	}
}

func (kc *KafkaConsumer) Close() error {
	if kc.reader != nil {
		return kc.reader.Close()
	}
	return nil
}
