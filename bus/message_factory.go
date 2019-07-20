package bus

import "github.com/google/uuid"

func GenerateRequest(channel string, payload interface{}) *Message {
    return &Message {
        Id: uuid.New(),
        Channel: channel,
        Payload: payload,
        Direction: Request }
}

func GenerateResponse(channel string, payload interface{}) *Message {
    return &Message {
        Id: uuid.New(),
        Channel: channel,
        Payload: payload,
        Direction: Response }
}
