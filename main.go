package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	config := SQSConfiguration{
		QueueName:           "fc-poc",
		MaxNumberOfMessages: 10,
		WaitTime:            1,
		Region:              "ap-southeast-2",
	}

	sqs, err := NewSQSService(&config)
	if err != nil {
		panic(err)
	}

	dispatcher := NewHttpDispatcher("http://localhost:9000/")
	mp := NewMessagePump(sqs, dispatcher)
	mp.Start()
	log.Info("Message pump started")
	err = <-mp.Done
}
