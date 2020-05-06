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

package kafka

import (
	"blaster/core"
	"context"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

type Binder struct {
	Group         sarama.ConsumerGroup
	KafkaConfig   *Config
	Config        *core.Config
	runner        core.MessagePumpRunner
	awaiter       *core.Awaiter
	awaitNotifier *core.AwaitNotifier
	logFields     log.Fields
}

func (b *Binder) Start(ctx context.Context) {
	go func() {
		defer b.Group.Close()

		// Iterate over consumer sessions until we have an error
		// or the context is cancelled.
		for {
			topics := []string{b.KafkaConfig.Topic}
			handler := &SaramaConsumerGroupHandler{
				Binding:         b,
				Context:         ctx,
				PartionHandlers: make(map[int32]*PartionHandler),
				done:            make(chan struct{}),
				logFields:       log.Fields{"module": "consumer_group_handler"},
			}

			err := b.Group.Consume(ctx, topics, handler)
			if err != nil {
				b.awaitNotifier.Notify(err)
				return
			}

			// Wait for the cleanup of current handler before starting a new one.
			// TODO: Consider if this can be done in the background.
			<-handler.Done()

			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			select {
			case <-ctx.Done():
				b.awaitNotifier.Notify(nil)
				return
			default:
			}
		}
	}()
}

func (b *Binder) Awaiter() *core.Awaiter {
	return b.awaiter
}

type KafkaBinderBuilder struct {
}

func (b *KafkaBinderBuilder) Build(runner core.MessagePumpRunner, coreConfig *core.Config, options interface{}) (core.BrokerBinder, error) {
	kafkaConfig := options.(Config)
	config := sarama.NewConfig()
	config.Version = sarama.V2_4_0_0 // specify appropriate version
	config.Consumer.Return.Errors = true
	if kafkaConfig.StartFromOldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	if kafkaConfig.BufferSize != 0 {
		config.ChannelBufferSize = kafkaConfig.BufferSize
	}

	g, err := sarama.NewConsumerGroup(kafkaConfig.BrokerAddresses, kafkaConfig.Group, config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	logFields := log.Fields{"module": "kafka_binding"}
	// Track errors
	go func() {
		for err := range g.Errors() {
			log.WithFields(log.Fields{"err": err}).Info("error in consumer group")
		}
	}()

	awaiter, awaitNotifier := core.NewAwaiter()
	return &Binder{
		Group:         g,
		KafkaConfig:   &kafkaConfig,
		Config:        coreConfig,
		runner:        runner,
		awaiter:       awaiter,
		awaitNotifier: awaitNotifier,
		logFields:     logFields,
	}, nil
}
