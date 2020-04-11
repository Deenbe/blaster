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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSuccessfulNotification(t *testing.T) {
	awaiter, awaitNotifier := NewAwaiter()

	go func() {
		awaitNotifier.Notify(errors.New("doh"))
	}()

	assert.EqualError(t, awaiter.Err(), "doh")
}

func TestDoneChannel(t *testing.T) {
	awaiter, awaitNotifier := NewAwaiter()

	go func() {
		awaitNotifier.Notify(nil)
	}()

	<-awaiter.Done()
}
