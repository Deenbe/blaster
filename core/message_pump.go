/*
Copyright © 2020 Blaster Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"context"
	"runtime"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var EmptyMessageSet []*Message = []*Message{}

type Message struct {
	MessageID  string                 `json:"messageId"`
	Body       string                 `json:"body"`
	Properties map[string]interface{} `json:"properties"`
	Data       map[string]interface{} `json:"-"`
}

// Transporter is the common interface used to interact with
// an underlaying broker.
type Transporter interface {
	Messages() <-chan []*Message
	Delete(*Message) error
	Poison(*Message) error
}

// Dispatcher is the interface used to deliver the message
// to the handler process.
type Dispatcher interface {
	Dispatch(*Message) error
}

// BrokerBinder is how we glue the machinery of message
// pump and handler manager integration to a particular broker
// with rest of the system.
// A BrokerBinder must have a BrokerBinderBuilder counterpart with
// initialisation logic. BrokerBinder is responsible for initialising
// the appropriate Transporter and executing the MessagePumpRunner
// with desired configuration.
type BrokerBinder interface {
	Start(context.Context)
	Awaiter() *Awaiter
}

// MessagePumpRunner is the interface between a BrokerBinder and
// the core plumbing of MessagePump and HandlerManager pair.
// BrokerBinders can use Run method to execute an instance of
// the pair with specified configuration.
type MessagePumpRunner interface {
	Run(context.Context, Transporter, Config) *Awaiter
}

// BrokerBinderBuilder constructs a BrokerBinder
type BrokerBinderBuilder interface {
	Build(MessagePumpRunner, *Config, interface{}) (BrokerBinder, error)
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
	Transporter   Transporter
	Dispatcher    Dispatcher
	RetryPolicy   *RetryPolicy
	MaxHandlers   int
	DispatchDone  chan struct{}
	awaiter       *Awaiter
	awaitNotifier *AwaitNotifier
	logFields     log.Fields
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
				buf, ok := <-p.Transporter.Messages()
				if !ok {
					p.awaitNotifier.Notify(errors.WithStack(errors.New("transporter closed")))
					return
				}
				buffer = buf
				log.WithFields(p.logFields).Debugf("received %d messages", len(buffer))
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
					p.awaitNotifier.Notify(ctx.Err())
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
					p.awaitNotifier.Notify(ctx.Err())
					return
				default:
				}
			}
		}
	}()
}

func (p *MessagePump) Awaiter() *Awaiter {
	return p.awaiter
}

func (p *MessagePump) dispatch(message *Message, sw *Stopwatch) {
	defer func() {
		p.DispatchDone <- struct{}{}
		log.WithFields(p.logFields).WithFields(log.Fields{"messageId": message.MessageID, "duration": sw.Total(), "duration_parts": sw.Laps}).Info("dispatched")
	}()

	sw.Lap("scheduled")
	e := p.RetryPolicy.Execute(func() error {
		return p.Dispatcher.Dispatch(message)
	}, "dispatch message %s", message.MessageID)

	sw.Lap("handler-invoked")
	if e != nil {
		log.WithFields(p.logFields).Infof("message_pump: failed to dispatch message %s\n", message.MessageID)
		e = p.Transporter.Poison(message)
		sw.Lap("poisoned")
		if e != nil {
			log.WithFields(p.logFields).Infof("message_pump: failed to poison message %s\n", message.MessageID)
		}
		return
	}

	e = p.RetryPolicy.Execute(func() error {
		return p.Transporter.Delete(message)
	}, "delete message %s", message.MessageID)

	sw.Lap("deleted")
	if e != nil {
		log.WithFields(p.logFields).Infof("message_pump: failed to delete message %s\n", message.MessageID)
	}
}

func NewMessagePump(transporter Transporter, dispatcher Dispatcher, retryCount int, retryDelay time.Duration, maxHandlers int) *MessagePump {
	if maxHandlers == 0 {
		maxHandlers = runtime.NumCPU() * 256
	}
	logFields := log.Fields{"module": "message_pump"}
	awaiter, awaitNotifier := NewAwaiter()
	return &MessagePump{
		Transporter:   transporter,
		Dispatcher:    dispatcher,
		RetryPolicy:   NewRetryPolicy(retryCount, retryDelay),
		DispatchDone:  make(chan struct{}, maxHandlers),
		MaxHandlers:   maxHandlers,
		awaiter:       awaiter,
		awaitNotifier: awaitNotifier,
		logFields:     logFields,
	}
}

type DefaultMessagePumpRunner struct {
	logFields log.Fields
}

func (r *DefaultMessagePumpRunner) Run(ctx context.Context, transporter Transporter, config Config) *Awaiter {
	awaiter, awaitNotifier := NewAwaiter()
	go func() {
		dispatcher := NewHttpDispatcher(config.HandlerURL)
		pump := NewMessagePump(transporter, dispatcher, config.RetryCount, config.RetryDelay, config.MaxHandlers)
		hm := NewHandlerManager(config.HandlerCommand, config.HandlerArgs, config.HandlerURL, config.StartupDelaySeconds)
		hm.Start(ctx)
		pump.Start(ctx)

		select {
		case <-hm.Awaiter.Done():
		case <-pump.Awaiter().Done():
		}

		err := hm.Awaiter.Err()
		log.WithFields(r.logFields).WithField("err", err).Info("handler manager exited")

		err = pump.Awaiter().Err()
		log.WithFields(r.logFields).WithField("err", err).Info("message pump exited")

		awaitNotifier.Notify(errors.New("default mesage pump runner exited"))
	}()
	return awaiter
}

func NewDefaultMessagePumpRunner() *DefaultMessagePumpRunner {
	return &DefaultMessagePumpRunner{
		logFields: log.Fields{"module": "default_message_pump_runner"},
	}
}
