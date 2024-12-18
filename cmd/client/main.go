package main

import (
	//"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	// Create new game state
	gameState := gamelogic.NewGameState(username)

	// Set up signal channel for Ctrl+C
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start the REPL loop
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			if err := gameState.CommandSpawn(words); err != nil {
				fmt.Println("Error:", err)
			}
		case "move":
			if _, err := gameState.CommandMove(words); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Move command executed successfully.")
			}
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}

		select {
		case <-signalChan:
			fmt.Println("\nReceived interrupt signal, exiting...")
			return
		default:
		}
	}
}
