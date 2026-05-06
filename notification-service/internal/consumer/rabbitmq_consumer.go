package consumer

import (
	"encoding/json"
	"log"
	"notification-service/internal/domain"
	"notification-service/internal/usecase"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	mainQueue   = "payment.completed"
	dlxExchange = "payment.dlx"
	dlqQueue    = "payment.completed.dlq"
	maxRetries  = 3
)

type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	notifier *usecase.Notifier
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

	return &RabbitMQConsumer{conn: conn, channel: ch, notifier: notifier}, nil
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

func retryCount(msg amqp.Delivery) int64 {
	xDeath, ok := msg.Headers["x-death"]
	if !ok {
		return 0
	}
	deaths, ok := xDeath.([]interface{})
	if !ok || len(deaths) == 0 {
		return 0
	}
	entry, ok := deaths[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, _ := entry["count"].(int64)
	return count
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
			log.Printf("[Consumer] Malformed message, sending to DLQ: %v", err)
			msg.Nack(false, false)
			continue
		}

		retries := retryCount(msg)

		if event.Amount == 13 {
			if retries >= maxRetries {
				log.Printf("[DLQ] Order %s failed %d times → moving to DLQ", event.OrderID, retries)
				msg.Nack(false, false)
			} else {
				log.Printf("[RETRY] Order %s failed (attempt %d/%d), requeueing...", event.OrderID, retries+1, maxRetries)
				msg.Nack(false, true)
			}
			continue
		}

		c.notifier.Handle(event)
		msg.Ack(false)
	}

	return nil
}

func (c *RabbitMQConsumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
