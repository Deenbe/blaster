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
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type CommitRequest struct {
	done    chan error
	Message *core.Message
}

func (r *CommitRequest) Complete(err error) {
	r.done <- err
	close(r.done)
}

type BufferedCommitter struct {
	commitFunc               func(*sarama.ConsumerMessage)
	config                   *Config
	numberOfMessagesObserved int
	lastSeenMessage          *core.Message
	commitRequests           chan *CommitRequest
	lastCommitTime           time.Time
	after                    <-chan time.Time
	now                      func() time.Time
	awaiter                  *core.Awaiter
	awaitNotifier            *core.AwaitNotifier
	logFields                log.Fields
	dispatcher               core.Dispatcher
}

func getConsumerMessage(msg *core.Message) *sarama.ConsumerMessage {
	return msg.Data[DataItemMessage].(*sarama.ConsumerMessage)
}

func getLatestMessage(a, b *core.Message) *core.Message {
	if a == nil {
		return b
	}

	if b == nil {
		return a
	}

	cma := getConsumerMessage(a)
	cmb := getConsumerMessage(b)
	if cma.Offset < cmb.Offset {
		return b
	}
	return a
}

func (c *BufferedCommitter) Start() {
	go func() {
		for {
			select {
			case now := <-c.after:
				if (now.Sub(c.lastCommitTime)) < (time.Second * time.Duration(c.config.CommitIntervalSeconds)) {
					continue
				}
				if c.lastSeenMessage == nil {
					continue
				}
				c.commitAndReset(c.lastSeenMessage, CommitReasonIdleTimeout)
			case req, ok := <-c.commitRequests:
				if !ok {
					c.awaitNotifier.Notify(nil)
					return
				}

				c.numberOfMessagesObserved++
				// Ignore if this message offset is earlier than the last seen one.
				// Although, this should not happen in practice.
				msg := getLatestMessage(req.Message, c.lastSeenMessage)
				c.lastSeenMessage = msg
				if c.numberOfMessagesObserved < c.config.CommitBatchSize {
					req.Complete(nil)
					continue
				}

				err := c.commitAndReset(msg, CommitReasonBufferFull)
				req.Complete(err)
			}
		}
	}()
}

func (c *BufferedCommitter) Commit(msg *core.Message) error {
	req := &CommitRequest{
		done:    make(chan error),
		Message: msg,
	}

	c.commitRequests <- req
	return <-req.done
}

func (c *BufferedCommitter) Stop() {
	close(c.commitRequests)
}

func (c *BufferedCommitter) Awaiter() *core.Awaiter {
	return c.awaiter
}

func (c *BufferedCommitter) commitAndReset(msg *core.Message, reason string) error {
	if c.config.PreCommitHookPath != "" {
		err := c.dispatcher.Dispatch(msg)
		if err != nil {
			log.WithFields(c.logFields).WithField("err", err).Info("pre-commit hook failed")
			return err
		}
	}

	cm := getConsumerMessage(msg)
	c.commitFunc(cm)
	log.WithFields(c.logFields).WithFields(log.Fields{
		"partion":        cm.Partition,
		"offset":         cm.Offset,
		"reason":         reason,
		"bufferSize":     c.numberOfMessagesObserved,
		"lastCommitTime": c.lastCommitTime,
	}).Info("offset committed")
	c.numberOfMessagesObserved = 0
	c.lastCommitTime = c.now()
	c.lastSeenMessage = nil
	return nil
}

func NewBuferedComitter(config *Config, now func() time.Time, after <-chan time.Time, dispatcher core.Dispatcher, commitFunc func(*sarama.ConsumerMessage)) *BufferedCommitter {
	awaiter, awaitNotifier := core.NewAwaiter()
	return &BufferedCommitter{
		lastCommitTime: now(),
		commitFunc:     commitFunc,
		awaiter:        awaiter,
		awaitNotifier:  awaitNotifier,
		config:         config,
		commitRequests: make(chan *CommitRequest),
		now:            now,
		after:          after,
		logFields:      log.Fields{"module": "buferred_committer"},
		dispatcher:     dispatcher,
	}
}
