package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/1729prashant/learn-pub-sub-starter/internal/gamelogic"
	"github.com/1729prashant/learn-pub-sub-starter/internal/pubsub"
	"github.com/1729prashant/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		gs.HandlePause(ps) // Pass the entire PlayingState struct
		if ps.IsPaused {
			fmt.Println("Game paused.")
		} else {
			fmt.Println("Game resumed.")
		}
		defer fmt.Print("> ")
		return pubsub.Ack // Always acknowledge pause/resume messages
	}
}

func handlerMove(gs *gamelogic.GameState, conn *amqp.Connection) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		outcome, err := gs.HandleMove(am, conn)
		if err != nil {
			fmt.Printf("Error handling move: %v\n", err)
			return pubsub.NackRequeue // Requeue on any error during move handling
		}
		defer fmt.Print("> ")

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			// If there's no error publishing the war declaration, we acknowledge the move
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		outcome, _, _ := gs.HandleWar(rw)
		defer fmt.Print("> ")

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon, gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			fmt.Println("Unexpected war outcome, discarding message.")
			return pubsub.NackDiscard
		}
	}
}

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

	// Subscribe to pause/resume messages
	pauseQueueName := routing.PauseKey + "." + username
	err = pubsub.SubscribeJSON[routing.PlayingState](
		conn,
		routing.ExchangePerilDirect,
		pauseQueueName,
		routing.PauseKey,
		routing.TransientQueue,
		handlerPause(gamelogic.NewGameState(username)),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to subscribe to pause messages: %s", err))
	}

	// Create new game state
	gameState := gamelogic.NewGameState(username)

	// Subscribe to other players' moves
	moveQueueName := routing.ArmyMovesPrefix + "." + username
	err = pubsub.SubscribeJSON[gamelogic.ArmyMove](
		conn,
		routing.ExchangePerilTopic,
		moveQueueName,
		routing.ArmyMovesPrefix+".*",
		routing.TransientQueue,
		handlerMove(gameState, conn),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to subscribe to army moves: %s", err))
	}

	// Subscribe to war recognitions
	err = pubsub.SubscribeJSON[gamelogic.RecognitionOfWar](
		conn,
		routing.ExchangePerilTopic,
		"war",                              // Queue name is just "war"
		routing.WarRecognitionsPrefix+".*", // Matches all war recognition messages
		routing.DurableQueue,               // Use a durable queue
		handlerWar(gameState),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to subscribe to war recognitions: %s", err))
	}

	// Create a new channel for publishing
	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Failed to open a channel: %s", err))
	}
	defer ch.Close()

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
			armyMove, err := gameState.CommandMove(words)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				// Publish the move
				err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, armyMove)
				if err != nil {
					fmt.Printf("Failed to publish move: %s\n", err)
				} else {
					fmt.Println("Move published successfully.")
				}
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
