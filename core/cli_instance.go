package core

import (
	"context"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
)

func RunCLIInstance(binding BrokerBinder, config *Config) error {
	if config.EnableVersboseLog {
		log.SetLevel(log.DebugLevel)
	}

	ctx := context.Background()
	cancelCtx, cancelFunc := context.WithCancel(ctx)

	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, os.Interrupt)

	binding.Start(cancelCtx)

	select {
	case <-binding.Awaiter().Done():
	case <-chanSignal:
	}

	cancelFunc()
	err := binding.Awaiter().Err()
	log.WithFields(log.Fields{"module": "cli_instance", "err": err}).Info("cli instance exited")
	return err
}
