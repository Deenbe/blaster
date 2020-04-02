package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	done         chan error
}

func (h *HandlerManager) Start(ctx context.Context) {
	go func() {
		url, err := url.Parse(h.HandlerURL)
		if err != nil {
			h.close(err)
			return
		}
		cmd := exec.CommandContext(ctx, h.Command, h.Argv...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("BLASTER_HANDLER_PORT=%v", url.Port()))
		err = cmd.Run()
		if err != nil {
			log.WithFields(log.Fields{"module": "handler_manager", "err": err}).Info("error handler process existed")
		}
		h.close(err)
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

func (h *HandlerManager) Done() <-chan error {
	return h.done
}

func (h *HandlerManager) close(err error) {
	h.done <- err
	close(h.done)
}

func NewHandlerManager(command string, argv []string, handlerURL string, startupDelaySeconds int) *HandlerManager {
	return &HandlerManager{
		Command:      command,
		Argv:         argv,
		HandlerURL:   handlerURL,
		StartupDelay: time.Second * time.Duration(startupDelaySeconds),
		done:         make(chan error),
	}
}
