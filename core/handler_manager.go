package core

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

type HandlerManager struct {
	Command      string
	Argv         []string
	StartupDelay time.Duration
	HandlerURL   string
	Done         chan error
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

	if h.StartupDelay != 0 {
		log.WithFields(log.Fields{"module": "handler_manager"}).Infof("waiting for handler to start (delay is %v)", h.StartupDelay)
		time.Sleep(h.StartupDelay)
		log.WithFields(log.Fields{"module": "handler_manager"}).Info("resuming now")
		return
	}

	// Probe the readiness endpoint for 5 times
	// TODO: make this configurable
	log.WithFields(log.Fields{"module": "handler_manager"}).Info("startup delay is disabled. probing readiness endpoint.")
	for i := 0; i < 5; i++ {
		if i != 0 {
			time.Sleep(time.Second * time.Duration(i))
		}
		response, e := http.Get(h.HandlerURL)
		if e != nil {
			log.WithFields(log.Fields{"module": "handler_manager", "error": e}).Infof("probing attempt %d failed", i)
			continue
		}
		defer response.Body.Close()
		if response.StatusCode == 200 {
			log.WithFields(log.Fields{"module": "handler_manager"}).Info("handler is ready")
			return
		}
		if response.StatusCode == 404 || response.StatusCode == 501 {
			log.WithFields(log.Fields{"module": "handler_manager", "status code": response.StatusCode}).Info("probing endpoint is not available")
			return
		}
	}
	log.WithFields(log.Fields{"module": "handler_manager"}).Warn("handler readiness is inconclusive")
}

func NewHandlerManager(command string, argv []string, handlerURL string, startupDelaySeconds int) *HandlerManager {
	return &HandlerManager{
		Command:      command,
		Argv:         argv,
		Done:         make(chan error),
		HandlerURL:   handlerURL,
		StartupDelay: time.Second * time.Duration(startupDelaySeconds),
	}
}
