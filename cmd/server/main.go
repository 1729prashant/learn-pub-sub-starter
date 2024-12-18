package main

import (
	"fmt"
	"log"

	"github.com/1729prashant/learn-pub-sub-starter/internal/gamelogic"
	"github.com/1729prashant/learn-pub-sub-starter/internal/pubsub"
	"github.com/1729prashant/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const AMQP_URL = "amqp://guest:guest@localhost:5672/"

func main() {
	// Connect to RabbitMQ server
	conn, err := amqp.Dial(AMQP_URL)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to RabbitMQ: %s", err))
	}
	defer conn.Close()

	// Print connection success message
	fmt.Println("Successfully connected to RabbitMQ")

	// Create a new channel
	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Failed to open a channel: %s", err))
	}
	defer ch.Close()

	// Declare and bind a durable queue to the peril_topic exchange
	_, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", routing.DurableQueue)
	if err != nil {
		log.Fatalf("Failed to declare and bind queue: %s", err)
	}
	fmt.Printf("Queue %s declared and bound successfully.\n", q.Name)

	// Print help information
	gamelogic.PrintServerHelp()

	for {
		// Get user input
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			fmt.Println("Sending pause message...")
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				fmt.Printf("Failed to publish pause message: %s\n", err)
			}
		case "resume":
			fmt.Println("Sending resume message...")
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				fmt.Printf("Failed to publish resume message: %s\n", err)
			}
		case "quit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Command not understood.")
		}
	}
}
