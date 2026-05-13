package publisher

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	mainQueue   = "payment.completed"
	dlxExchange = "payment.dlx"
	dlqQueue    = "payment.completed.dlq"
)

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type Publisher interface {
	Publish(event PaymentEvent) error
	Close()
}

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
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
	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
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

func (p *RabbitMQPublisher) Publish(event PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(
		context.Background(),
		"",
		mainQueue,
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (p *RabbitMQPublisher) Close() {
	p.channel.Close()
	p.conn.Close()
}
