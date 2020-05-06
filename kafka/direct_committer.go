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

	"github.com/Shopify/sarama"
)

type DirectCommitter struct {
	commitFunc func(*sarama.ConsumerMessage)
	awaiter    *core.Awaiter
}

func (c *DirectCommitter) Start() {}
func (c *DirectCommitter) Stop()  {}
func (c *DirectCommitter) Awaiter() *core.Awaiter {
	return c.awaiter
}

func (c *DirectCommitter) Commit(m *core.Message) error {
	c.commitFunc(getConsumerMessage(m))
	return nil
}

func NewDirectCommitter(commitFunc func(*sarama.ConsumerMessage)) *DirectCommitter {
	awaiter, notifier := core.NewAwaiter()
	notifier.Notify(nil)
	return &DirectCommitter{
		commitFunc: commitFunc,
		awaiter:    awaiter,
	}
}
