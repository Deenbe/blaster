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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSuccessfulExecution(t *testing.T) {
	count := 0
	p := NewRetryPolicy(1, time.Millisecond)
	err := p.Execute(func() error {
		count = count + 1
		return nil
	}, "test")

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRetryAndSuccess(t *testing.T) {
	count := 0
	p := NewRetryPolicy(1, time.Millisecond)
	err := p.Execute(func() error {
		count = count + 1
		if count == 1 {
			return errors.New("doh")
		}
		return nil
	}, "test")

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRetryAndFail(t *testing.T) {
	count := 0
	p := NewRetryPolicy(1, time.Microsecond)
	err := p.Execute(func() error {
		count = count + 1
		return errors.New("doh")
	}, "test")

	assert.Error(t, err, "doh")
	assert.Equal(t, 2, count)
}
