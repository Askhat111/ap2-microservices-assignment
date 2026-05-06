package consumer

import (
	"encoding/json"
	"log"
	"notification-service/internal/domain"
	"notification-service/internal/usecase"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	mainQueue   = "payment.completed"
	dlxExchange = "payment.dlx"
	dlqQueue    = "payment.completed.dlq"
	maxRetries  = 3
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

	if err := setupQueues(ch); err != nil {
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

func setupQueues(ch *amqp.Channel) error {
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

func (c *RabbitMQConsumer) getAndIncrementRetry(eventID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := c.retryCounts[eventID]
	c.retryCounts[eventID]++
	return count
}

func (c *RabbitMQConsumer) clearRetry(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.retryCounts, eventID)
}

func (c *RabbitMQConsumer) publishToDLQ(body []byte) error {
	return c.channel.Publish(
		dlxExchange,
		dlqQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (c *RabbitMQConsumer) Start() error {
	msgs, err := c.channel.Consume(
		mainQueue, "notification-service",
		false, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	log.Println("[Consumer] Waiting for payment events...")

	for msg := range msgs {
		var event domain.PaymentEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("[Consumer] Malformed message, discarding: %v", err)
			msg.Ack(false)
			continue
		}

		if event.Amount == 13 {
			attempt := c.getAndIncrementRetry(event.EventID)

			if attempt >= maxRetries {
				log.Printf("[DLQ] Order %s failed %d times → moving to DLQ", event.OrderID, attempt)
				c.publishToDLQ(msg.Body)
				c.clearRetry(event.EventID)
				msg.Ack(false)
			} else {
				log.Printf("[RETRY] Order %s failed (attempt %d/%d), requeueing...", event.OrderID, attempt+1, maxRetries)
				msg.Nack(false, true)
			}
			continue
		}

		c.notifier.Handle(event)
		c.clearRetry(event.EventID)
		msg.Ack(false)
	}

	return nil
}

func (c *RabbitMQConsumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
