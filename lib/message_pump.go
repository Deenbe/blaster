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
	RetryCount   int
	RetryDelay   time.Duration
}

func (p *MessagePump) Start() {
	go func() {
		for {
			msgs, err := p.QueueService.Read()
			if err != nil {
				p.Done <- err
			}

			log.Infof("message_pump: received %d messages\n", len(msgs))
			for _, msg := range msgs {
				go func() {
					e := p.executeWithRetry(msg, "dispatch", func(m *Message) error {
						return p.Dispatcher.Dispatch(m)
					})
					if e != nil {
						log.Warnf("message_pump: failed to dispatch message %s\n", msg.MessageID)
					}
					e = p.executeWithRetry(msg, "delete", func(m *Message) error {
						return p.QueueService.Delete(m)
					})
					if e != nil {
						log.Warnf("message_pump: failed to delete message %s\n", msg.MessageID)
					}
				}()
			}
		}
	}()
}

func (p *MessagePump) executeWithRetry(m *Message, name string, op func(*Message) error) error {
	err := op(m)
	if err == nil {
		return nil
	}

	for i := 0; i < p.RetryCount; i++ {
		time.Sleep(p.RetryDelay)
		err = op(m)
		if err != nil {
			// TODO: Do structured logging
			log.Warningf("message_pump: error in attempt %d to %s message %s\n", i, name, m.MessageID)
			continue
		}
	}
	return err
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher, retryCount int, retryDelay time.Duration) *MessagePump {
	return &MessagePump{queueReader, dispatcher, make(chan error), retryCount, retryDelay}
}
