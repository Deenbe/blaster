package lib

import (
	"context"
	"os"
	"os/signal"
	"runtime"
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
	MaxHandlers  int
	DispatchDone chan struct{}
}

// Start the main message pump loop.
// This process consists of following steps:
// - Read some messages from the broker
// - Dispatch as many as possible respecting MaxHandlers setting
// - Exit if context is cancelled or wait
// 	 until a handler returns if we have exhausted MaxHandlers
func (p *MessagePump) Start(ctx context.Context) {
	go func() {
		activeHandlers := 0
		buffer := []*Message{}
		for {
			// First we need to fill the buffer with some messages.
			// We use buffer length to indicate whether we have messages
			// read during the previous iteration but not yet dispatched due
			// to MaxHandlers limit.
			if len(buffer) == 0 {
				var err error
				buffer, err = p.QueueService.Read()
				if err != nil {
					log.WithFields(log.Fields{
						"module": "message_pump",
						"error":  err,
					}).Info("message_pump: failed to read from queue")
				} else {
					log.Infof("messge_pump: received %d messages", len(buffer))
				}
			}

			// Workout how many messages can be dispatched
			// based on MaxHandlers setting and dispatch that amount.
			numberOfMessagesToDispatch := len(buffer)
			numberOfAvailableHandlers := p.MaxHandlers - activeHandlers
			if numberOfMessagesToDispatch > numberOfAvailableHandlers {
				numberOfMessagesToDispatch = numberOfAvailableHandlers
			}
			for _, m := range buffer[:numberOfMessagesToDispatch] {
				// TODO: Propagate ctx so that these go rotines
				// can be notified of cancellation
				go p.dispatch(m)
			}

			buffer = buffer[numberOfMessagesToDispatch:]
			activeHandlers += numberOfMessagesToDispatch

			if activeHandlers == p.MaxHandlers {
				// We have exhausted the MaxHandlers,
				// we should wait until one of them is released or
				// the context is cancelled.
				select {
				case <-ctx.Done():
					p.close(ctx.Err())
					return
				case <-p.DispatchDone:
					activeHandlers--
				}
			} else {
				// We have room for running more handlers.
				// If context is not cancelled, proceed to the next iteration
				// of the loop with a default case.
				select {
				case <-ctx.Done():
					p.close(ctx.Err())
					return
				default:
				}
			}
		}
	}()
}

func (p *MessagePump) dispatch(message *Message) {
	defer func() { p.DispatchDone <- struct{}{} }()

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
	log.WithFields(log.Fields{
		"module": "message_pump",
		"error":  err,
	}).Infof("message_pump: stopped")
	p.Done <- err
	close(p.Done)
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher, retryCount int, retryDelay time.Duration, maxHandlers int) *MessagePump {
	if maxHandlers == 0 {
		maxHandlers = runtime.NumCPU() * 256
	}
	return &MessagePump{
		QueueService: queueReader,
		Dispatcher:   dispatcher,
		RetryPolicy:  NewRetryPolicy(retryCount, retryDelay),
		Done:         make(chan error),
		DispatchDone: make(chan struct{}, maxHandlers),
		MaxHandlers:  maxHandlers,
	}
}

func StartTheSystem(messagePump *MessagePump, handlerName string, handlerArgv []string) error {
	var err error
	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, os.Interrupt)

	messagePump.Start(cancelCtx)

	h := NewHandlerManager(handlerName, handlerArgv)
	h.Start(cancelCtx)

	select {
	case err = <-messagePump.Done:
	case err = <-h.Done:
	case <-chanSignal:
	}

	cancelFunc()
	<-messagePump.Done
	<-h.Done

	return err
}
