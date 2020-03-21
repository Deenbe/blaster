package lib

import (
	"time"

	log "github.com/sirupsen/logrus"
)

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
	RetryPolicy  *RetryPolicy
}

func (p *MessagePump) Start() {
	go func() {
		for {
			msgs, err := p.QueueService.Read()
			if err != nil {
				// TODO: Add an error specialisation to
				// to demarcate the errors that should stop the pump
				p.Done <- err
			}

			log.Infof("message_pump: received %d messages\n", len(msgs))
			for _, msg := range msgs {
				go func() {
					e := p.RetryPolicy.Execute(func() error {
						return p.Dispatcher.Dispatch(msg)
					}, "dispatch message %s", *msg.MessageID)

					if e != nil {
						log.Infof("message_pump: failed to dispatch message %s\n", *msg.MessageID)
					}

					e = p.RetryPolicy.Execute(func() error {
						return p.QueueService.Delete(msg)
					}, "delete message %s", *msg.MessageID)

					if e != nil {
						log.Infof("message_pump: failed to delete message %s\n", *msg.MessageID)
					}
				}()
			}
		}
	}()
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher, retryCount int, retryDelay time.Duration) *MessagePump {
	return &MessagePump{queueReader, dispatcher, make(chan error), NewRetryPolicy(retryCount, retryDelay)}
}
