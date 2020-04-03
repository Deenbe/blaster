package core

import (
	"context"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
)

func RunCLIInstance(binding BrokerBinding, config *Config) error {
	if config.EnableVersboseLog {
		log.SetLevel(log.DebugLevel)
	}

	var err error
	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, os.Interrupt)

	binding.Start(cancelCtx)

	select {
	case err = <-binding.Done():
	case <-chanSignal:
	}

	cancelFunc()
	<-binding.Done()
	return err
}
