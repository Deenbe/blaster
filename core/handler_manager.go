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
	Command       string
	Argv          []string
	StartupDelay  time.Duration
	HandlerURL    string
	Awaiter       *Awaiter
	awaitNotifier *AwaitNotifier
	logFields     log.Fields
}

func (h *HandlerManager) Start(ctx context.Context) {
	go func() {
		url, err := url.Parse(h.HandlerURL)
		if err != nil {
			h.awaitNotifier.Notify(err)
			return
		}
		cmd := exec.CommandContext(ctx, h.Command, h.Argv...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("BLASTER_HANDLER_PORT=%v", url.Port()))
		err = cmd.Run()
		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
		}
		log.WithFields(h.logFields).WithField("err", err).Infof("handler process (%d) exited", pid)
		h.awaitNotifier.Notify(err)
	}()

	if h.StartupDelay != 0 {
		log.WithFields(h.logFields).Infof("waiting for handler to start (delay is %v)", h.StartupDelay)
		time.Sleep(h.StartupDelay)
		log.WithFields(h.logFields).Info("resuming now")
		return
	}

	// Probe the readiness endpoint for 5 times
	// TODO: make this configurable
	log.WithFields(h.logFields).Info("startup delay is disabled. probing readiness endpoint.")
	for i := 0; i < 5; i++ {
		if i != 0 {
			time.Sleep(time.Second * time.Duration(i))
		}
		response, e := http.Get(h.HandlerURL)
		if e != nil {
			log.WithFields(h.logFields).WithField("err", e).Infof("probing attempt %d failed", i)
			continue
		}
		defer response.Body.Close()
		if response.StatusCode == 200 {
			log.WithFields(h.logFields).Info("handler is ready")
			return
		}
		if response.StatusCode == 404 || response.StatusCode == 501 {
			log.WithFields(h.logFields).WithField("status code", response.StatusCode).Info("probing endpoint is not available")
			return
		}
	}
	log.WithFields(h.logFields).Warn("handler readiness is inconclusive")
}

func NewHandlerManager(command string, argv []string, handlerURL string, startupDelaySeconds int) *HandlerManager {
	logFields := log.Fields{"module": "handler_manager"}
	awaiter, awaitNotifier := NewAwaiter()
	return &HandlerManager{
		Command:       command,
		Argv:          argv,
		HandlerURL:    handlerURL,
		StartupDelay:  time.Second * time.Duration(startupDelaySeconds),
		Awaiter:       awaiter,
		awaitNotifier: awaitNotifier,
		logFields:     logFields,
	}
}
