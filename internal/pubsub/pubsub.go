package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/1729prashant/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishJSON publishes a JSON-encoded message to a RabbitMQ exchange using the given channel.
func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	// Marshal the value to JSON
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Publish message to exchange
	err = ch.PublishWithContext(
		context.Background(),
		exchange, // exchange
		key,      // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonBytes,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}
	return nil
}

// DeclareAndBind declares and binds a queue to an exchange in RabbitMQ
func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	// After connecting to RabbitMQ
	fmt.Println("Successfully connected to RabbitMQ")

	durable := simpleQueueType == routing.DurableQueue
	autoDelete := simpleQueueType == routing.TransientQueue
	exclusive := simpleQueueType == routing.TransientQueue

	// Before declaring queue
	fmt.Printf("Attempting to declare queue with name: %s\n", queueName)
	fmt.Printf("Exchange: %s, RoutingKey: %s\n", routing.ExchangePerilDirect, routing.PauseKey)

	q, err := ch.QueueDeclare(
		queueName,  // name
		durable,    // durable
		autoDelete, // autoDelete
		exclusive,  // exclusive
		false,      // noWait
		nil,        // args
	)
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}

	// After declaring queue
	fmt.Printf("Queue declared with properties: name=%s, messages=%d, consumers=%d\n", q.Name, q.Messages, q.Consumers)

	fmt.Printf("Queue details - Name: %s, Durable: %v, AutoDelete: %v, Exclusive: %v\n", q.Name, durable, autoDelete, exclusive)

	err = ch.QueueBind(
		q.Name,   // queue name
		key,      // routing key
		exchange, // exchange
		false,    // noWait
		nil,      // args
	)
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}

	return ch, q, nil
}
