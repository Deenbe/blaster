package core

import (
	log "github.com/sirupsen/logrus"
)

// AwaitNotifier counterpart of an `Awaiter`
type AwaitNotifier struct {
	done chan struct{}
	err  error
}

// Notify sets the provided error message
// as current goroutines exit reason and
// signals the `Awaiter`.
func (n *AwaitNotifier) Notify(err error) {
	n.err = err
	close(n.done)
}

// Awaiter is a coordination primitive used to
// signal the completion of an independently executing
// goroutine.
// When goroutine A needs to make its completion observable
// to B, A should create an Awaiter and its AwaitNotifier counterpart.
// Then it hands Awaiter reference to B. B can then use either
// `Done()` channel or `Err()` method to wait for A's completion.
type Awaiter struct {
	notifier *AwaitNotifier
	fields   log.Fields
}

// Done channel is used to wait until the `Awaiter`
// is signaled. It can be used in `select` statements
// perform this as a non-blocking action.
func (a *Awaiter) Done() <-chan struct{} {
	return a.notifier.done
}

// Err blocks until the `Awaiter` signaled and
// returns the error if available.
func (a *Awaiter) Err() error {
	<-a.Done()
	return a.notifier.err
}

// NewAwaiter creates a new `Awaiter` and `AwaitNotifier`
// pair.
func NewAwaiter() (*Awaiter, *AwaitNotifier) {
	notifier := &AwaitNotifier{
		done: make(chan struct{}),
	}

	return &Awaiter{notifier: notifier}, notifier
}
