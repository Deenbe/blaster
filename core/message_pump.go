package core

import (
	"context"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

type Message struct {
	MessageID  string                 `json:"messageId"`
	Body       string                 `json:"body"`
	Properties map[string]interface{} `json:"properties"`
	Data       map[string]interface{} `jason:-`
}

type QueueService interface {
	Read() ([]*Message, error)
	Delete(*Message) error
	Poison(*Message) error
}

type Dispatcher interface {
	Dispatch(*Message) error
}

// BrokerBinding is how we glue the machinery of message
// pump and handler manager integration to a particular broker
// with rest of the system.
type BrokerBinding interface {
	Start(context.Context)
	Done() <-chan error
}

// Config of common knobs.
type Config struct {
	RetryCount          int
	RetryDelay          time.Duration
	MaxHandlers         int
	HandlerURL          string
	HandlerCommand      string
	HandlerArgs         []string
	StartupDelaySeconds int
	EnableVersboseLog   bool
}

type MessagePump struct {
	QueueService QueueService
	Dispatcher   Dispatcher
	RetryPolicy  *RetryPolicy
	done         chan error
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
			// Use buffer length to indicate whether we have messages
			// read during the previous iteration but not yet dispatched due
			// to MaxHandlers limit.
			if len(buffer) == 0 {
				var err error
				buffer, err = p.QueueService.Read()
				if err != nil {
					log.WithFields(log.Fields{"module": "message_pump", "error": err}).Info("message_pump: failed to read from queue")
				} else {
					log.Debugf("messge_pump: received %d messages", len(buffer))
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
				go p.dispatch(m, NewStopwatch())
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

func (p *MessagePump) dispatch(message *Message, sw *Stopwatch) {
	defer func() {
		p.DispatchDone <- struct{}{}
		log.WithFields(log.Fields{"messageId": message.MessageID, "duration": sw.Total(), "duration_parts": sw.Laps}).Info("dispatched")
	}()

	sw.Lap("scheduled")
	e := p.RetryPolicy.Execute(func() error {
		return p.Dispatcher.Dispatch(message)
	}, "dispatch message %s", message.MessageID)

	sw.Lap("handler-invoked")
	if e != nil {
		log.Infof("message_pump: failed to dispatch message %s\n", message.MessageID)
		e = p.QueueService.Poison(message)
		sw.Lap("poisoned")
		if e != nil {
			log.Infof("message_pump: failed to poison message %s\n", message.MessageID)
		}
		return
	}

	e = p.RetryPolicy.Execute(func() error {
		return p.QueueService.Delete(message)
	}, "delete message %s", message.MessageID)

	sw.Lap("deleted")
	if e != nil {
		log.Infof("message_pump: failed to delete message %s\n", message.MessageID)
	}
}

func (p *MessagePump) Done() <-chan error {
	return p.done
}

func (p *MessagePump) close(err error) {
	log.WithFields(log.Fields{"module": "message_pump", "error": err}).Infof("message_pump: stopped")
	p.done <- err
	close(p.done)
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher, retryCount int, retryDelay time.Duration, maxHandlers int) *MessagePump {
	if maxHandlers == 0 {
		maxHandlers = runtime.NumCPU() * 256
	}
	return &MessagePump{
		QueueService: queueReader,
		Dispatcher:   dispatcher,
		RetryPolicy:  NewRetryPolicy(retryCount, retryDelay),
		DispatchDone: make(chan struct{}, maxHandlers),
		MaxHandlers:  maxHandlers,
		done:         make(chan error),
	}
}
