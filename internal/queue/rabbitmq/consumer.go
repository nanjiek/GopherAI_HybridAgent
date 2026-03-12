package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/internal/core/service"
	"gophermind/internal/obs/metrics"
	"gophermind/pkg/contracts/events"
)

const (
	consumerName = "task_consumer"
)

// Consumer implements task_queue consumption with retry/DLQ and idempotency.
type Consumer struct {
	conn      *amqp091.Connection
	ch        *amqp091.Channel
	taskQueue string
	maxRetry  int
	producer  service.QueueProducer
	cache     service.SessionCache
	inboxRepo service.ConsumerInboxRepository
	logger    *zap.Logger
}

// NewConsumer creates a RabbitMQ consumer.
func NewConsumer(cfg config.RabbitMQConfig, cache service.SessionCache, inboxRepo service.ConsumerInboxRepository, producer service.QueueProducer, logger *zap.Logger) (*Consumer, error) {
	conn, err := amqp091.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := ch.Qos(16, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(cfg.TaskQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &Consumer{
		conn:      conn,
		ch:        ch,
		taskQueue: cfg.TaskQueue,
		maxRetry:  cfg.MaxRetry,
		producer:  producer,
		cache:     cache,
		inboxRepo: inboxRepo,
		logger:    logger,
	}, nil
}

// Start continuously consumes tasks and applies retry/DLQ policy.
func (c *Consumer) Start(ctx context.Context) error {
	deliveries, err := c.ch.Consume(
		c.taskQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("rabbitmq deliveries channel closed")
			}
			if err := c.handleDelivery(ctx, d); err != nil {
				requeue := shouldRequeue(err)
				if c.logger != nil {
					c.logger.Warn("consume task failed", zap.Bool("requeue", requeue), zap.Error(err))
				}
				_ = d.Nack(false, requeue)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, d amqp091.Delivery) error {
	var msg events.TaskMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		// Poison payload should be moved to DLQ and then acked to avoid infinite loops.
		if publishErr := c.producer.PublishDLQTask(ctx, poisonPayloadMessage(d, err)); publishErr != nil {
			return requeueErr(fmt.Errorf("publish poison payload to dlq failed: %w", publishErr))
		}
		metrics.IncMQDLQ()
		return nil
	}
	if msg.IdempotencyKey == "" {
		msg.IdempotencyKey = fallbackMessageID(msg)
	}

	idempotent, err := c.cache.IsIdempotent(ctx, consumerName, msg.IdempotencyKey)
	if err != nil {
		return requeueErr(err)
	}
	if idempotent {
		metrics.IncMQIdempotentHit()
		if c.logger != nil {
			c.logger.Info("skip duplicated task message", zap.String("request_id", msg.RequestID))
		}
		return nil
	}

	shouldProcess, err := c.inboxRepo.BeginProcess(ctx, consumerName, msg.IdempotencyKey, msg.RetryCount)
	if err != nil {
		return requeueErr(err)
	}
	if !shouldProcess {
		metrics.IncMQIdempotentHit()
		if err := c.cache.MarkIdempotent(ctx, consumerName, msg.IdempotencyKey, 48*time.Hour); err != nil && c.logger != nil {
			c.logger.Warn("mark idempotent cache failed", zap.Error(err))
		}
		return nil
	}

	if err := c.processBusiness(ctx, msg); err != nil {
		retryCount := msg.RetryCount + 1
		msg.RetryCount = retryCount
		msg.LastError = err.Error()

		if retryCount <= c.maxRetry && isRetryableError(err) {
			if publishErr := c.producer.PublishRetryTask(ctx, msg); publishErr != nil {
				return requeueErr(fmt.Errorf("publish retry task failed: %w", publishErr))
			}
			if markErr := c.inboxRepo.MarkFailed(ctx, consumerName, msg.IdempotencyKey, retryCount, err.Error()); markErr != nil && c.logger != nil {
				c.logger.Warn("mark inbox failed status failed", zap.Error(markErr))
			}
			metrics.IncMQRetry()
			return nil
		}

		if publishErr := c.producer.PublishDLQTask(ctx, msg); publishErr != nil {
			return requeueErr(fmt.Errorf("publish task to dlq failed: %w", publishErr))
		}
		if markErr := c.inboxRepo.MarkDead(ctx, consumerName, msg.IdempotencyKey, retryCount, err.Error()); markErr != nil && c.logger != nil {
			c.logger.Warn("mark inbox dead failed", zap.Error(markErr))
		}
		metrics.IncMQDLQ()
		return nil
	}

	if err := c.inboxRepo.MarkSucceeded(ctx, consumerName, msg.IdempotencyKey); err != nil {
		return requeueErr(err)
	}
	if err := c.cache.MarkIdempotent(ctx, consumerName, msg.IdempotencyKey, 48*time.Hour); err != nil && c.logger != nil {
		c.logger.Warn("mark idempotent cache failed", zap.Error(err))
	}
	return nil
}

func (c *Consumer) processBusiness(ctx context.Context, msg events.TaskMessage) error {
	// Keep minimal business processing: publish task result to result_queue.
	res := events.ResultMessage{
		EventType:      "task.processed",
		Version:        "v1",
		JobID:          msg.JobID,
		IdempotencyKey: msg.IdempotencyKey,
		RequestID:      msg.RequestID,
		SessionID:      msg.SessionID,
		UserID:         msg.UserID,
		Status:         "ok",
		Provider:       msg.ModelType,
		TraceID:        msg.TraceID,
		CreatedAt:      time.Now(),
	}
	return c.producer.PublishResult(ctx, res)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var syntaxErr *json.SyntaxError
	return !errors.As(err, &syntaxErr)
}

func fallbackMessageID(msg events.TaskMessage) string {
	switch {
	case msg.RequestID != "":
		return msg.RequestID
	case msg.JobID != "":
		return msg.JobID
	default:
		return uuid.NewString()
	}
}

func poisonPayloadMessage(d amqp091.Delivery, unmarshalErr error) events.TaskMessage {
	id := d.MessageId
	if id == "" {
		id = uuid.NewString()
	}
	return events.TaskMessage{
		EventType:      "task.invalid_payload",
		Version:        "v1",
		JobID:          id,
		IdempotencyKey: id,
		RequestID:      id,
		RetryCount:     0,
		LastError:      unmarshalErr.Error(),
		CreatedAt:      time.Now(),
	}
}

type deliveryError struct {
	err     error
	requeue bool
}

func (e *deliveryError) Error() string {
	return e.err.Error()
}

func (e *deliveryError) Unwrap() error {
	return e.err
}

func requeueErr(err error) error {
	return &deliveryError{err: err, requeue: true}
}

func shouldRequeue(err error) bool {
	var de *deliveryError
	if errors.As(err, &de) {
		return de.requeue
	}
	return false
}

// Close releases amqp resources.
func (c *Consumer) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
