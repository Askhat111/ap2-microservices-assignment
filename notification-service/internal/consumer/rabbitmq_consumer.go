package consumer

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"
)

const (
	mainQueue     = "payment.completed"
	dlxExchange   = "payment.dlx"
	dlqQueue      = "payment.completed.dlq"
	dlqMaxRetries = 3
)

type RabbitMQConsumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	notifier    *usecase.Notifier
	mu          sync.Mutex
	retryCounts map[string]int
}

func NewRabbitMQConsumer(url string, notifier *usecase.Notifier) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := declareQueues(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &RabbitMQConsumer{
		conn:        conn,
		channel:     ch,
		notifier:    notifier,
		retryCounts: make(map[string]int),
	}, nil
}

func declareQueues(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(dlxExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(dlqQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(dlqQueue, dlqQueue, dlxExchange, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		mainQueue, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    dlxExchange,
			"x-dead-letter-routing-key": dlqQueue,
		},
	); err != nil {
		return err
	}
	return nil
}

func (c *RabbitMQConsumer) dlqAttempt(eventID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := c.retryCounts[eventID]
	c.retryCounts[eventID]++
	return count
}

func (c *RabbitMQConsumer) clearDLQCount(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.retryCounts, eventID)
}

func (c *RabbitMQConsumer) sendToDLQ(body []byte) {
	err := c.channel.Publish(dlxExchange, dlqQueue, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		log.Printf("[Consumer] Failed to publish to DLQ: %v", err)
	}
}

func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		mainQueue, "notification-service",
		false, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	log.Println("[Worker] Background worker started, waiting for events...")

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			c.handle(ctx, msg)
		}
	}
}

func (c *RabbitMQConsumer) handle(ctx context.Context, msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] Malformed message, discarding: %v", err)
		msg.Ack(false)
		return
	}

	if event.Amount == 13 {
		attempt := c.dlqAttempt(event.EventID)
		if attempt >= dlqMaxRetries {
			log.Printf("[DLQ] Event %s failed %d times → moving to DLQ", event.EventID, attempt)
			c.sendToDLQ(msg.Body)
			c.clearDLQCount(event.EventID)
			msg.Ack(false)
		} else {
			log.Printf("[DLQ-RETRY] Event %s attempt %d/%d, requeueing...",
				event.EventID, attempt+1, dlqMaxRetries)
			msg.Nack(false, true)
		}
		return
	}

	c.notifier.Handle(ctx, event)
	c.clearDLQCount(event.EventID)
	msg.Ack(false)
}

func (c *RabbitMQConsumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
