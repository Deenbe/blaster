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

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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
