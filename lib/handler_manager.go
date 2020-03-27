package lib

import (
	"context"
	"os"
	"os/exec"
	"time"
)

type HandlerManager struct {
	Command string
	Argv    []string
	Done    chan error
}

func (h *HandlerManager) Start(ctx context.Context, delay time.Duration) {
	go func() {
		cmd := exec.CommandContext(ctx, h.Command, h.Argv...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		h.Done <- cmd.Run()
		close(h.Done)
	}()
	// We give 5 seconds for the handler to start although this is
	// not very reliable.
	// TODO: Change this to probe a readiness endpoint in the target process
	time.Sleep(delay)
}

func NewHandlerManager(command string, argv []string) *HandlerManager {
	return &HandlerManager{
		Command: command,
		Argv:    argv,
		Done:    make(chan error),
	}
}
