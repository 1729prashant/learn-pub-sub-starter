package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

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
	fmt.Printf("Exchange: %s, RoutingKey: %s\n", exchange, key) // Updated to use generic exchange and key

	// Define the arguments for the queue declaration, including the dead letter exchange
	args := amqp.Table{}
	if queueName != "war" && queueName != "game_logs" { // Exclude both 'war' and 'game_logs' queues from DLX setup
		args["x-dead-letter-exchange"] = "peril_dlx"
	}

	q, err := ch.QueueDeclare(
		queueName,  // name
		durable,    // durable
		autoDelete, // autoDelete
		exclusive,  // exclusive
		false,      // noWait
		args,       // args - now includes dead letter exchange
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

// AckType represents the acknowledgment types for message handling.
type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

// subscribe is a helper function for both JSON and GOB subscriptions
func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	// Ensure the queue exists and is bound to the exchange
	ch, q, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("failed to declare and bind queue: %v", err)
	}

	// Create a new channel for consuming messages
	deliveries, err := ch.Consume(
		q.Name, // queue
		"",     // consumer tag - empty string for auto-generated
		false,  // auto-ack - set to false because we'll manually acknowledge
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %v", err)
	}

	// Start a goroutine to process messages
	go func() {
		for d := range deliveries {
			var msg T
			if msg, err = unmarshaller(d.Body); err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				d.Nack(false, true)
				log.Println("Message NackRequeue due to unmarshal error")
				continue
			}

			// Call the handler function with the unmarshaled message and get the AckType
			ackType := handler(msg)

			switch ackType {
			case Ack:
				d.Ack(false)
				log.Println("Message Acknowledged")
			case NackRequeue:
				d.Nack(false, true)
				log.Println("Message NackRequeue")
			case NackDiscard:
				d.Nack(false, false)
				log.Println("Message NackDiscard")
			default:
				d.Ack(false)
				log.Println("Unexpected AckType, Message Acknowledged")
			}
		}
	}()

	return nil
}

// SubscribeJSON uses JSON unmarshalling
func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, func(data []byte) (T, error) {
		var result T
		if err := json.Unmarshal(data, &result); err != nil {
			return result, err
		}
		return result, nil
	})
}

// SubscribeGob uses GOB unmarshalling
func SubscribeGob(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
	handler func(routing.GameLog) AckType,
) error {
	return subscribe[routing.GameLog](conn, exchange, queueName, key, simpleQueueType, handler, func(data []byte) (routing.GameLog, error) {
		var result routing.GameLog
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&result); err != nil {
			return result, err
		}
		return result, nil
	})
}

// PublishGob publishes a GOB-encoded message to a RabbitMQ exchange using the given channel.
func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	// Encode the value to GOB
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(val); err != nil {
		return fmt.Errorf("failed to encode to GOB: %v", err)
	}

	// Publish GOB message to exchange
	err := ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        buf.Bytes(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}
	return nil
}
