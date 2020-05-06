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

const DataItemMessage string = "message"
const DefaultCommitIntervalSeconds int = 15
const CommitReasonBufferFull string = "buffer-full"
const CommitReasonIdleTimeout string = "idle-timeout"

type PartionHandler struct {
	Transporter   *Transporter
	RunnerAwaiter *core.Awaiter
	Committer     Committer
	Started       bool
}

type Committer interface {
	Start()
	Stop()
	Commit(*core.Message) error
	Awaiter() *core.Awaiter
}
