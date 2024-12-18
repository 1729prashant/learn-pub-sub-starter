package main

import (
	"fmt"
	"os"

	"github.com/1729prashant/learn-pub-sub-starter/internal/gamelogic"
	"github.com/1729prashant/learn-pub-sub-starter/internal/pubsub"
	"github.com/1729prashant/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	// Prompt user for username
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to RabbitMQ: %s", err))
	}
	defer conn.Close()

	// Declare and bind a transient queue
	queueName := routing.PauseKey + "." + username
	ch, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, routing.TransientQueue)
	if err != nil {
		panic(fmt.Sprintf("Failed to declare and bind queue: %s", err))
	}
	defer ch.Close()

	fmt.Printf("Queue %s declared and bound successfully.\n", q.Name)
	// Here you would typically start consuming messages or continue with other client logic
}
