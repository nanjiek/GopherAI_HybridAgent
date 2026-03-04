package rabbitmq

import (
	"github.com/nanjiek/GopherAI_HybridAgent/config"
	"fmt"
	"log"
	"strconv"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection

const maxRetryCount = 3

func initConn() {
	c := config.GetConfig()
	mqURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		c.RabbitmqUsername, c.RabbitmqPassword, c.RabbitmqHost, c.RabbitmqPort, c.RabbitmqVhost,
	)
	var err error
	conn, err = amqp.Dial(mqURL)
	if err != nil {
		log.Fatalf("RabbitMQ connection failed: %v", err)
	}
}

type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	Exchange string
	Key      string
}

func NewRabbitMQ(exchange string, key string) *RabbitMQ {
	return &RabbitMQ{Exchange: exchange, Key: key}
}

func (r *RabbitMQ) Destroy() {
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}

func NewWorkRabbitMQ(queue string) *RabbitMQ {
	rabbitmq := NewRabbitMQ("", queue)

	if conn == nil {
		initConn()
	}
	rabbitmq.conn = conn

	var err error
	rabbitmq.channel, err = rabbitmq.conn.Channel()
	if err != nil {
		panic(err.Error())
	}

	if err = rabbitmq.channel.Qos(10, 0, false); err != nil {
		panic(err.Error())
	}

	if err = rabbitmq.declareQueues(); err != nil {
		panic(err.Error())
	}

	return rabbitmq
}

func (r *RabbitMQ) declareQueues() error {
	dlq := r.Key + ".dlq"
	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": dlq,
	}

	if _, err := r.channel.QueueDeclare(r.Key, true, false, false, false, args); err != nil {
		return err
	}
	if _, err := r.channel.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) Publish(message []byte) error {
	if err := r.declareQueues(); err != nil {
		return err
	}

	return r.channel.Publish(
		r.Exchange,
		r.Key,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         message,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (r *RabbitMQ) Consume(handle func(msg *amqp.Delivery) error) {
	if err := r.declareQueues(); err != nil {
		panic(err)
	}

	msgs, err := r.channel.Consume(r.Key, "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	for msg := range msgs {
		if err := handle(&msg); err != nil {
			if retryErr := r.retryMessage(msg); retryErr != nil {
				log.Printf("[rabbitmq] retry failed: %v", retryErr)
				_ = msg.Nack(false, false)
				continue
			}
			_ = msg.Ack(false)
			continue
		}
		_ = msg.Ack(false)
	}
}

func (r *RabbitMQ) retryMessage(msg amqp.Delivery) error {
	currentRetry := readRetryCount(msg.Headers)
	if currentRetry >= maxRetryCount {
		return fmt.Errorf("retry count exceeded: %d", currentRetry)
	}

	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = currentRetry + 1

	return r.channel.Publish(
		r.Exchange,
		r.Key,
		false,
		false,
		amqp.Publishing{
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			Headers:      headers,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func readRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	v, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, err := strconv.Atoi(t)
		if err == nil {
			return n
		}
	}
	return 0
}
