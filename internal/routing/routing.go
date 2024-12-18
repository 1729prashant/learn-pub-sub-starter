package routing

const (
	ArmyMovesPrefix = "army_moves"

	WarRecognitionsPrefix = "war"

	PauseKey = "pause"

	GameLogSlug = "game_logs"
)

const (
	ExchangePerilDirect = "peril_direct"
	ExchangePerilTopic  = "peril_topic"
)

// QueueType enum for use in pubsub package
const (
	DurableQueue int = iota
	TransientQueue
)
