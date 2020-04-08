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
