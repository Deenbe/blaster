// +build test
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

package utils

import (
	"time"
)

type TestClock struct {
	now time.Time
}

func (c *TestClock) Now() time.Time {
	return c.now
}

func (c *TestClock) Forward(d time.Duration) {
	c.now = c.now.Add(d)
}

func (c *TestClock) Rewind(d time.Duration) {
	c.now = c.now.Add(d * -1)
}

func NewTestClock() *TestClock {
	return &TestClock{
		now: time.Date(2000, 01, 01, 00, 00, 00, 00, time.Local),
	}
}
