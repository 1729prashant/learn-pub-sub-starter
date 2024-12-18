package main

import (
	"fmt"
	"log"

	// "time"

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

	// Print help information
	gamelogic.PrintServerHelp()

	// Subscribe to game logs with SubscribeGob
	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,      // Assuming game_logs is the queue name
		routing.GameLogSlug+".*", // Wildcard to capture all logs
		routing.DurableQueue,
		func(gl routing.GameLog) pubsub.AckType {
			defer fmt.Print("> ") // New prompt after handling the log
			log.Printf("Received GameLog: %+v", gl)

			// Write log to disk
			if err := gamelogic.WriteLog(gl); err != nil {
				log.Printf("Failed to write log: %v\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		},
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to subscribe to game logs: %s", err))
	}

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
