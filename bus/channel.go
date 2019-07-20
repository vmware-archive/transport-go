package bus

type Channel struct {
    Name string `json:"string"`
    Stream chan *Message
    initialized bool
}

func (channel *Channel) Boot() {
    channel.Stream = make(chan *Message)
}

func (channel *Channel) Send(message *Message) {
    if !channel.initialized {
        channel.Boot()
    }
    go func() {
        channel.Stream <- message
    }()
}
