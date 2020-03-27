package lib

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecution(t *testing.T) {
	h := NewHandlerManager("echo", []string{"hey"})
	h.Start(context.Background(), 0)
	err := <-h.Done
	assert.NoError(t, err)
}

func TestCancelleation(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	h := NewHandlerManager("sleep", []string{"10"})
	h.Start(ctx, 0)
	cancelFunc()
	err := <-h.Done
	assert.Error(t, err, "context canceled")
}
