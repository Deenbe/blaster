package lib

type Message struct {
	MessageID  *string                `json:"messageId"`
	Body       *string                `json:"body"`
	Properties map[string]interface{} `json:"properties"`
}

type QueueService interface {
	Read() ([]*Message, error)
	Delete(*Message) error
}

type Dispatcher interface {
	Dispatch(*Message) error
}

type MessagePump struct {
	QueueService QueueService
	Dispatcher   Dispatcher
	Done         chan error
}

func (p *MessagePump) Start() {
	go func() {
		for {
			msgs, err := p.QueueService.Read()
			if err != nil {
				p.Done <- err
			}

			for _, msg := range msgs {
				go func() {
					e := p.Dispatcher.Dispatch(msg)
					if e != nil {
						p.Done <- e
						return
					}
					e = p.QueueService.Delete(msg)
					if e != nil {
						p.Done <- e
						return
					}
				}()
			}
		}
	}()
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher) *MessagePump {
	return &MessagePump{queueReader, dispatcher, make(chan error)}
}
