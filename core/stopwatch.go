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

import "time"

type Lap struct {
	Name     string
	Duration time.Duration
}

type Stopwatch struct {
	Laps      []Lap
	StartTime time.Time
	LapStart  time.Time
}

func (s *Stopwatch) Lap(name string) {
	s.Laps = append(s.Laps, Lap{name, time.Since(s.LapStart)})
	s.LapStart = time.Now()
}

func (s *Stopwatch) Total() time.Duration {
	return time.Since(s.StartTime)
}

func NewStopwatch() *Stopwatch {
	n := time.Now()
	return &Stopwatch{
		StartTime: n,
		LapStart:  n,
	}
}
