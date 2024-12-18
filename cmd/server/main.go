package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	// Wait for interrupt signal (Ctrl+C) to exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Print shutdown message and close connection (if not already closed by defer)
	fmt.Println("Shutting down the program")
	conn.Close()
}

/*
Declares the RabbitMQ connection string (AMQP_URL).
Uses amqp.Dial to attempt a connection to RabbitMQ. If the connection fails, it panics with an error message.
Defers the closing of the connection to ensure it's closed when the function exits.
Prints a message indicating that the connection was successful.
Sets up a channel (signalChan) to receive interrupt signals (like Ctrl+C or SIGTERM).
Waits for a signal to be received on signalChan. When a signal is received, it moves on to the next steps.
Prints a message indicating the program is shutting down.
Closes the connection explicitly, although defer would handle this if the program exits abruptly. This explicit close is included for clarity and to ensure the message about shutting down is printed before the connection is closed.
*/
