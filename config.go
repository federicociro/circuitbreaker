package main

type Mode int

const (
	ModeFail Mode = iota
	ModeQueue
	ModeQueuePeerInitiated
	ModeBlock
	ModeQueueLimited
)

func (m Mode) String() string {
	switch m {
	case ModeFail:
		return "FAIL"

	case ModeQueue:
		return "QUEUE"

	case ModeQueuePeerInitiated:
		return "QUEUE_PEER_INITIATED"

	case ModeBlock:
		return "BLOCK"

	case ModeQueueLimited:
		return "QUEUE_LIMITED"

	default:
		panic("unknown mode")
	}
}
