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
	"fmt"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type SaramaConsumerGroupHandler struct {
	PartionHandlers map[int32]*PartionHandler
	Mutex           sync.Mutex
	Binding         *Binder
	Context         context.Context
	done            chan struct{}
	logFields       log.Fields
}

func (h *SaramaConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	for _, partions := range session.Claims() {
		for _, p := range partions {
			var committer Committer
			// Make a copy of the config so that we can use it to
			// call MessagePumpRunner
			config := *h.Binding.Config
			port := core.GetFreePort()
			config.HandlerURL = fmt.Sprintf("http://localhost:%d/", port)
			dispatcher := core.NewHttpDispatcher(fmt.Sprintf("%s%s", config.HandlerURL, h.Binding.KafkaConfig.PreCommitHookPath))
			commitFunc := func(m *sarama.ConsumerMessage) {
				session.MarkMessage(m, "")
			}

			if h.Binding.KafkaConfig.CommitBatchSize > 0 {
				ticker := time.NewTicker(time.Second * time.Duration(h.Binding.KafkaConfig.CommitIntervalSeconds))
				committer = NewBuferedComitter(h.Binding.KafkaConfig, time.Now, ticker.C, dispatcher, commitFunc)
			} else {
				committer = NewDirectCommitter(commitFunc)
			}

			committer.Start()
			transporter := &Transporter{
				messages:  make(chan []*core.Message),
				committer: committer,
			}

			ph := &PartionHandler{
				Transporter:   transporter,
				Committer:     committer,
				RunnerAwaiter: h.Binding.runner.Run(session.Context(), transporter, config),
			}

			h.PartionHandlers[p] = ph
		}
	}
	return nil
}

func (h *SaramaConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	for _, v := range h.PartionHandlers {
		if !v.Started {
			v.Transporter.Close()
		}
		err := v.RunnerAwaiter.Err()
		log.WithFields(h.logFields).WithFields(log.Fields{"generationId": session.GenerationID(), "err": err}).Info("message pump exited")
		v.Committer.Stop()
		<-v.Committer.Awaiter().Done()
	}
	close(h.done)
	log.WithFields(h.logFields).WithFields(log.Fields{"generationId": session.GenerationID()}).Info("consumer group handler is cleaned up")
	return nil
}

func (h *SaramaConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.Mutex.Lock()
	p, ok := h.PartionHandlers[claim.Partition()]
	if !ok {
		return errors.WithStack(errors.New("unable to consume a claim with an unclaimed partion"))
	}
	p.Started = true
	h.Mutex.Unlock()

	// Loop until the messages channel is open or
	// one of partion handler components exit.
	// When a partion handler component exit, we return
	// from the method and leave Sarama to cancel the
	// session. Since session's context is associated
	// with both MessagePump and HandlerManager, this
	// should gracefully shutdown all components.
ReceiveLoop:
	for {
		var m *core.Message
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				break ReceiveLoop
			}
			m = &core.Message{
				MessageID:  string(msg.Key),
				Body:       string(msg.Value),
				Properties: make(map[string]interface{}),
				Data:       make(map[string]interface{}),
			}
			m.Properties["timestamp"] = msg.Timestamp
			m.Properties["partitionId"] = msg.Partition
			m.Properties["offset"] = msg.Offset
			m.Data[DataItemMessage] = msg
		case <-p.RunnerAwaiter.Done():
			break ReceiveLoop
		}

		select {
		case p.Transporter.messages <- []*core.Message{m}:
		case <-p.RunnerAwaiter.Done():
			break ReceiveLoop
		}
	}
	p.Transporter.Close()
	return nil
}

func (h *SaramaConsumerGroupHandler) Done() <-chan struct{} {
	return h.done
}
