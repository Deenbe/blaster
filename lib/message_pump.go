package lib

import (
	"context"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
)

type Message struct {
	MessageID  string                 `json:"messageId"`
	Body       string                 `json:"body"`
	Properties map[string]interface{} `json:"properties"`
}

type QueueService interface {
	Read() ([]*Message, error)
	Delete(*Message) error
	Poison(*Message) error
}

type Dispatcher interface {
	Dispatch(*Message) error
}

type MessagePump struct {
	QueueService QueueService
	Dispatcher   Dispatcher
	RetryPolicy  *RetryPolicy
	Done         chan error
}

func (p *MessagePump) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.WithFields(log.Fields{
					"module": "message_pump",
					"error":  ctx.Err(),
				}).Infof("message_pump: stopped")
				p.close(nil)
				return
			default:
				p.pumpMain()
			}
		}
	}()
}

func (p *MessagePump) pumpMain() {
	messages, err := p.QueueService.Read()
	if err != nil {
		log.WithFields(log.Fields{
			"module": "message_pump",
			"error":  err,
		}).Info("message_pump: failed to read from queue")
		return
	}

	log.Infof("message_pump: received %d messages\n", len(messages))
	p.dispatchAll(messages)
}

func (p *MessagePump) dispatchAll(messages []*Message) {
	for _, m := range messages {
		go p.dispatch(m)
	}
}

func (p *MessagePump) dispatch(message *Message) {
	e := p.RetryPolicy.Execute(func() error {
		return p.Dispatcher.Dispatch(message)
	}, "dispatch message %s", message.MessageID)

	if e != nil {
		log.Infof("message_pump: failed to dispatch message %s\n", message.MessageID)
		e = p.QueueService.Poison(message)
		if e != nil {
			log.Infof("message_pump: failed to poison message %s\n", message.MessageID)
		}
		return
	}

	e = p.RetryPolicy.Execute(func() error {
		return p.QueueService.Delete(message)
	}, "delete message %s", message.MessageID)

	if e != nil {
		log.Infof("message_pump: failed to delete message %s\n", message.MessageID)
	}
}

func (p *MessagePump) close(err error) {
	p.Done <- err
	close(p.Done)
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher, retryCount int, retryDelay time.Duration) *MessagePump {
	return &MessagePump{queueReader, dispatcher, NewRetryPolicy(retryCount, retryDelay), make(chan error)}
}

func StartTheSystem(messagePump *MessagePump) error {
	var err error
	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, os.Interrupt)

	messagePump.Start(cancelCtx)

	select {
	case err = <-messagePump.Done:
	case <-chanSignal:
	}

	cancelFunc()
	<-messagePump.Done

	return err
}
