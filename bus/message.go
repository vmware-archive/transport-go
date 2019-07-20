package bus

import "github.com/google/uuid"

type Direction int

const (
    Request Direction = 0
    Response Direction = 1
    Error Direction = 2
)

type Message struct {
    Id uuid.UUID            `json:"id"`
    Channel string          `json:"channel"`
    Payload interface{}     `json:"payload"`
    Direction Direction     `json:"direction"`
}

