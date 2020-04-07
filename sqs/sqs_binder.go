package sqs

import (
	"blaster/core"
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type SQSConfiguration struct {
	QueueName           string
	Region              string
	MaxNumberOfMessages int64
	WaitTime            int64
}

type SQSTransporter struct {
	Configuration *SQSConfiguration
	Client        *sqs.SQS
	QueueUrl      string
}

func (t *SQSTransporter) Read() ([]*core.Message, error) {
	msgs, err := t.Client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            &t.QueueUrl,
		MaxNumberOfMessages: &t.Configuration.MaxNumberOfMessages,
		WaitTimeSeconds:     &t.Configuration.WaitTime,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := make([]*core.Message, len(msgs.Messages))

	for i, msg := range msgs.Messages {
		result[i] = &core.Message{
			MessageID: *msg.MessageId,
			Body:      *msg.Body,
			Properties: map[string]interface{}{
				"attributes":             msg.Attributes,
				"messageAttributes":      msg.MessageAttributes,
				"receiptHandle":          msg.ReceiptHandle,
				"md5OfBody":              msg.MD5OfBody,
				"md5OfMessageAttributes": msg.MD5OfMessageAttributes,
			},
		}
	}

	return result, nil
}

func (t *SQSTransporter) Delete(message *core.Message) error {
	rh, ok := message.Properties["receiptHandle"].(*string)
	if !ok {
		panic("Unexpected input: SQS message without a receipt handle")
	}

	_, err := t.Client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &t.QueueUrl,
		ReceiptHandle: rh,
	})

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (t *SQSTransporter) Poison(message *core.Message) error {
	return nil
}

func NewSQSTransporter(configuration *SQSConfiguration) (*SQSTransporter, error) {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(configuration.Region),
	}))

	svc := sqs.New(s)
	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(configuration.QueueName),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &SQSTransporter{
		Configuration: configuration,
		Client:        svc,
		QueueUrl:      *urlResult.QueueUrl,
	}, nil
}

type SQSBinder struct {
	SQSConfiguration *SQSConfiguration
	Config           *core.Config
	MessagePump      *core.MessagePump
	HandlerManager   *core.HandlerManager
	done             chan error
}

func (b *SQSBinder) Start(ctx context.Context) {
	go func() {
		b.HandlerManager.Start(ctx)
		b.MessagePump.Start(ctx)

		select {
		case b.done <- <-b.MessagePump.Done():
		case b.done <- <-b.HandlerManager.Done():
		}
		<-b.MessagePump.Done()
		<-b.HandlerManager.Done()
		close(b.done)
	}()
}

func (b *SQSBinder) Done() <-chan error {
	return b.done
}

func NewSQSBinder(sqsConfig *SQSConfiguration, config *core.Config) *SQSBinder {
	sqs, err := NewSQSTransporter(sqsConfig)
	if err != nil {
		panic(err)
	}
	dispatcher := core.NewHttpDispatcher(config.HandlerURL)
	mp := core.NewMessagePump(sqs, dispatcher, config.RetryCount, config.RetryDelay, config.MaxHandlers)
	hm := core.NewHandlerManager(config.HandlerCommand, config.HandlerArgs, config.HandlerURL, config.StartupDelaySeconds)
	return &SQSBinder{
		SQSConfiguration: sqsConfig,
		Config:           config,
		MessagePump:      mp,
		HandlerManager:   hm,
		done:             make(chan error),
	}
}
