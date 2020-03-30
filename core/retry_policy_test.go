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
