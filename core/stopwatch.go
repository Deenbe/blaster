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
