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
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type RetryPolicy struct {
	Count int
	Delay time.Duration
}

func (p *RetryPolicy) Execute(op func() error, idFormat string, args ...interface{}) error {
	err := op()
	if err == nil {
		return nil
	}

	id := fmt.Sprintf(idFormat, args...)

	log.WithFields(log.Fields{"module": "retry_policy", "operationId": id, "error": err}).Infof("retry_policy: operation %s failed. begin retrying\n", id)

	for i := 0; i < p.Count; i++ {
		time.Sleep(p.Delay)
		err = op()
		if err != nil {
			log.WithFields(log.Fields{"module": "retry_policy", "operationId": id, "error": err, "retryAttempt": i + 1}).Infof("retry_policy: operation %s failed.\n", id)
			continue
		}
	}
	return err
}

func NewRetryPolicy(count int, delay time.Duration) *RetryPolicy {
	return &RetryPolicy{count, delay}
}
