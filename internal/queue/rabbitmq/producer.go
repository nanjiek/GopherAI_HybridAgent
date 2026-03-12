package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/pkg/contracts/events"
)

// Producer 定义消息生产能力。
type Producer interface {
	PublishTask(ctx context.Context, message events.TaskMessage) error
	PublishResult(ctx context.Context, message events.ResultMessage) error
	PublishRetryTask(ctx context.Context, message events.TaskMessage) error
	PublishDLQTask(ctx context.Context, message events.TaskMessage) error
	Close() error
}

// AMQPProducer 是基于 RabbitMQ 的实现。
type AMQPProducer struct {
	conn        *amqp091.Connection
	ch          *amqp091.Channel
	taskQueue   string
	resultQueue string
	retryQueue1 string
	retryQueue2 string
	retryQueue3 string
	dlqQueue    string
	maxRetry    int
	logger      *zap.Logger
}

// NewProducer 初始化生产者并声明队列。
func NewProducer(cfg config.RabbitMQConfig, logger *zap.Logger) (*AMQPProducer, error) {
	conn, err := amqp091.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	// 主队列和结果队列
	if _, err = ch.QueueDeclare(cfg.TaskQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(cfg.ResultQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	// 重试队列：TTL 到期后回流 taskQueue。
	retryQueues := []struct {
		Name  string
		Delay int32
	}{
		{Name: cfg.RetryQueue1, Delay: int32(cfg.RetryDelay1.Milliseconds())},
		{Name: cfg.RetryQueue2, Delay: int32(cfg.RetryDelay2.Milliseconds())},
		{Name: cfg.RetryQueue3, Delay: int32(cfg.RetryDelay3.Milliseconds())},
	}
	for _, rq := range retryQueues {
		args := amqp091.Table{
			"x-message-ttl":             rq.Delay,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": cfg.TaskQueue,
		}
		if _, err = ch.QueueDeclare(rq.Name, true, false, false, false, args); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, err
		}
	}

	if _, err = ch.QueueDeclare(cfg.DLQQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &AMQPProducer{
		conn:        conn,
		ch:          ch,
		taskQueue:   cfg.TaskQueue,
		resultQueue: cfg.ResultQueue,
		retryQueue1: cfg.RetryQueue1,
		retryQueue2: cfg.RetryQueue2,
		retryQueue3: cfg.RetryQueue3,
		dlqQueue:    cfg.DLQQueue,
		maxRetry:    cfg.MaxRetry,
		logger:      logger,
	}, nil
}

// PublishTask 发布任务消息。
func (p *AMQPProducer) PublishTask(ctx context.Context, message events.TaskMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, "", p.taskQueue, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// PublishResult 发布结果消息。
func (p *AMQPProducer) PublishResult(ctx context.Context, message events.ResultMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, "", p.resultQueue, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// PublishRetryTask 发布到重试队列。
func (p *AMQPProducer) PublishRetryTask(ctx context.Context, message events.TaskMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	target := p.retryQueue1
	if message.RetryCount >= 2 {
		target = p.retryQueue2
	}
	if message.RetryCount >= 3 {
		target = p.retryQueue3
	}
	return p.ch.PublishWithContext(ctx, "", target, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// PublishDLQTask 发布到死信队列。
func (p *AMQPProducer) PublishDLQTask(ctx context.Context, message events.TaskMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, "", p.dlqQueue, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Close 关闭资源。
func (p *AMQPProducer) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// NoopProducer 是降级生产者。
type NoopProducer struct {
	logger *zap.Logger
}

// NewNoopProducer 构建无操作生产者。
func NewNoopProducer(logger *zap.Logger) *NoopProducer {
	return &NoopProducer{logger: logger}
}

// PublishTask 仅记录日志。
func (n *NoopProducer) PublishTask(_ context.Context, message events.TaskMessage) error {
	if n.logger != nil {
		n.logger.Debug("noop publish task", zap.String("job_id", message.JobID))
	}
	return nil
}

// PublishResult 仅记录日志。
func (n *NoopProducer) PublishResult(_ context.Context, message events.ResultMessage) error {
	if n.logger != nil {
		n.logger.Debug("noop publish result", zap.String("job_id", message.JobID))
	}
	return nil
}

// PublishRetryTask 仅记录日志。
func (n *NoopProducer) PublishRetryTask(_ context.Context, message events.TaskMessage) error {
	if n.logger != nil {
		n.logger.Debug("noop publish retry", zap.String("request_id", message.RequestID))
	}
	return nil
}

// PublishDLQTask 仅记录日志。
func (n *NoopProducer) PublishDLQTask(_ context.Context, message events.TaskMessage) error {
	if n.logger != nil {
		n.logger.Debug("noop publish dlq", zap.String("request_id", message.RequestID))
	}
	return nil
}

// Close 无需操作。
func (n *NoopProducer) Close() error {
	return nil
}
