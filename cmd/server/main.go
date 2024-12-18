package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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

	// Publish a pause message
	pauseState := routing.PlayingState{IsPaused: true}
	err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, pauseState)
	if err != nil {
		panic(fmt.Sprintf("Failed to publish message: %s", err))
	}

	// Wait for interrupt signal (Ctrl+C) to exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Print shutdown message and close connection (if not already closed by defer)
	fmt.Println("Shutting down the program")
	conn.Close()
}
