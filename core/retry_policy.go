package core

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type RetryPolicy struct {
	Count int
	Delay time.Duration
}

func (p *RetryPolicy) Execute(op func() error, idFormat string, args ...interface{}) error {
	err := op()
	if err == nil {
		return nil
	}

	id := fmt.Sprintf(idFormat, args...)

	log.WithFields(log.Fields{"module": "retry_policy", "operationId": id, "error": err}).Infof("retry_policy: operation %s failed. begin retrying\n", id)

	for i := 0; i < p.Count; i++ {
		time.Sleep(p.Delay)
		err = op()
		if err != nil {
			log.WithFields(log.Fields{"module": "retry_policy", "operationId": id, "error": err, "retryAttempt": i + 1}).Infof("retry_policy: operation %s failed.\n", id)
			continue
		}
	}
	return err
}

func NewRetryPolicy(count int, delay time.Duration) *RetryPolicy {
	return &RetryPolicy{count, delay}
}
