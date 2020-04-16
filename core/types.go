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
	"time"
)

var EmptyMessageSet []*Message = []*Message{}

// Message is the general purpose format used to
// transfer messages between the handler process
// and blaster.
type Message struct {
	MessageID  string                 `json:"messageId"`
	Body       string                 `json:"body"`
	Properties map[string]interface{} `json:"properties"`
	Data       map[string]interface{} `json:"-"`
}

// Transporter is the common interface used to interact with
// an underlaying broker. `BrokerBinder` is responsible for
// running the `MessagePump` with an appropriate implementation
// of this interface.
type Transporter interface {
	Messages() <-chan []*Message
	Delete(*Message) error
	Poison(*Message) error
}

// Dispatcher is the interface used to deliver the messages
// to the handler process.
type Dispatcher interface {
	Dispatch(*Message) error
}

// BrokerBinder is the glue between a `Transporter` and rest
// of the plumbing to deliver messages to handler process.
// It is responsible for initialising the appropriate
// Transporter and executing the `MessagePumpRunner`
// with desired configuration.
//
// A `BrokerBinder` can invoke `MessagePumpRunning` multiple
// times to start different instances of the delivery pipeline.
// For example `KafkaBrokerBinder` creates a new pipeline for
// each partion assigned to current blaster instance.
//
// A BrokerBinder must have a BrokerBinderBuilder counterpart with
// initialisation logic.
type BrokerBinder interface {
	Start(context.Context)
	Awaiter() *Awaiter
}

// MessagePumpRunner is the interface between a BrokerBinder and
// the core message delivery pipeline (i.e. `MessagePump`, `HandlerManager` and `Dispatcher`).
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
