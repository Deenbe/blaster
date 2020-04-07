package core

import (
	log "github.com/sirupsen/logrus"
)

type AwaitNotifier struct {
	done chan struct{}
	err  error
}

func (n *AwaitNotifier) Notify(err error) {
	n.err = err
	close(n.done)
}

type Awaiter struct {
	notifier *AwaitNotifier
	fields   log.Fields
}

func (a *Awaiter) Done() <-chan struct{} {
	return a.notifier.done
}

func (a *Awaiter) Err() error {
	<-a.Done()
	return a.notifier.err
}

func NewAwaiter() (*Awaiter, *AwaitNotifier) {
	notifier := &AwaitNotifier{
		done:   make(chan struct{}),
	}

	return &Awaiter{ notifier: notifier }, notifier
}
