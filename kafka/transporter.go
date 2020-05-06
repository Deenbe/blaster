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
)

type Transporter struct {
	messages  chan []*core.Message
	committer Committer
}

func (t *Transporter) Messages() <-chan []*core.Message {
	return t.messages
}

func (t *Transporter) Delete(m *core.Message) error {
	return t.committer.Commit(m)
}

func (t *Transporter) Poison(m *core.Message) error {
	return nil
}

func (t *Transporter) Close() {
	close(t.messages)
}
