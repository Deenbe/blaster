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
	"os"
	"os/signal"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func RunCLIInstance(bindingBuilder BrokerBinderBuilder, config *Config, options interface{}) error {
	if config.EnableVersboseLog {
		log.SetLevel(log.DebugLevel)
	}

	runner := NewDefaultMessagePumpRunner()
	binding, err := bindingBuilder.Build(runner, config, options)
	if err != nil {
		return errors.WithStack(err)
	}

	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, os.Interrupt)

	binding.Start(cancelCtx)

	select {
	case <-binding.Awaiter().Done():
	case <-chanSignal:
	}

	cancelFunc()
	err = binding.Awaiter().Err()
	log.WithFields(log.Fields{"module": "cli_instance", "err": err}).Info("cli instance exited")
	return err
}
