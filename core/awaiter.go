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

func (a *Awaiter) Wait() {
	<-a.Done()
	if a.notifier.err != nil {
		log.WithFields(a.fields).WithFields(log.Fields{"err": a.notifier.err}).Info("error module shutting down")
	}
}

func NewAwaiter(fields log.Fields) (*Awaiter, *AwaitNotifier) {
	notifier := &AwaitNotifier{
		done:   make(chan struct{}),
	}

	return &Awaiter{
		notifier: notifier,
		fields:   fields,
	}, notifier
}
