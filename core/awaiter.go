package core

import (
	log "github.com/sirupsen/logrus"
)

type AwaitNotifier struct {
	done chan struct{}
	errors chan error
}

func (n *AwaitNotifier) Notify(err error) {
	if err != nil {
		n.errors <- err
	}
	close(n.errors)
	close(n.done)
}

type Awaiter struct {
	Notifier *AwaitNotifier
	fields log.Fields
}

func (a *Awaiter) Done() <-chan struct{} {
	return a.Notifier.done
}

func (a *Awaiter) Wait() {
	<-a.Done()
	for err := range a.Notifier.done {
		log.WithFields(a.fields).WithFields(log.Fields{"err": err}).Info("error module shutting down")
	}
}

func NewAwaiter(fields log.Fields) (*Awaiter, *AwaitNotifier) {
	notifier := &AwaitNotifier{
		done: make(chan struct{}),
		errors: make(chan error, 1),
	}

	return &Awaiter{
		Notifier: notifier,
		fields: fields,
	}, notifier
}
