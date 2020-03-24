package lib

import (
	"context"
	"os"
	"os/exec"
)

type HandlerManager struct {
	Command string
	Argv    []string
	Done    chan error
}

func (h *HandlerManager) Start(ctx context.Context) {
	go func() {
		cmd := exec.CommandContext(ctx, h.Command, h.Argv...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		h.Done <- cmd.Run()
		close(h.Done)
	}()
}

func NewHandlerManager(command string, argv []string) *HandlerManager {
	return &HandlerManager{
		Command: command,
		Argv:    argv,
		Done:    make(chan error),
	}
}
