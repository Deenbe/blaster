package lib

import (
	"context"
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
		h.Done <- cmd.Run()
	}()
}

func NewHandlerManager(command string, argv []string) *HandlerManager {
	return &HandlerManager{
		Command: command,
		Argv:    argv,
		Done:    make(chan error),
	}
}
