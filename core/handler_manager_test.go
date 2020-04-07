package core

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const HandlerURL string = "http://localhost:8312/"

func TestExecution(t *testing.T) {
	done := make(chan struct{})
	server := createTestHandler(done)
	defer func() {
		server.Close()
		<-done
	}()

	h := NewHandlerManager("echo", []string{"hey"}, HandlerURL, 0)
	h.Start(context.Background())
	h.Awaiter.Err()
}

func TestCancelleation(t *testing.T) {
	done := make(chan struct{})
	server := createTestHandler(done)
	defer func() {
		server.Close()
		<-done
	}()

	ctx, cancelFunc := context.WithCancel(context.Background())
	h := NewHandlerManager("sleep", []string{"10"}, HandlerURL, 0)
	h.Start(ctx)
	cancelFunc()
	err := h.Awaiter.Err()
	assert.EqualError(t, err, "signal: killed")
}

func createTestHandler(done chan<- struct{}) *http.Server {
	server := &http.Server{
		Addr: ":8312",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	go func() {
		server.ListenAndServe()
		done <- struct{}{}
	}()

	return server
}
