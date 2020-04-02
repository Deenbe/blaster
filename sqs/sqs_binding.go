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

type SQSService struct {
	Configuration *SQSConfiguration
	Client        *sqs.SQS
	QueueUrl      string
}

func (s *SQSService) Read() ([]*core.Message, error) {
	msgs, err := s.Client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            &s.QueueUrl,
		MaxNumberOfMessages: &s.Configuration.MaxNumberOfMessages,
		WaitTimeSeconds:     &s.Configuration.WaitTime,
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

func (s *SQSService) Delete(message *core.Message) error {
	rh, ok := message.Properties["receiptHandle"].(*string)
	if !ok {
		panic("Unexpected input: SQS message without a receipt handle")
	}

	_, err := s.Client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &s.QueueUrl,
		ReceiptHandle: rh,
	})

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *SQSService) Poison(message *core.Message) error {
	return nil
}

func NewSQSService(configuration *SQSConfiguration) (*SQSService, error) {
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

	return &SQSService{
		Configuration: configuration,
		Client:        svc,
		QueueUrl:      *urlResult.QueueUrl,
	}, nil
}

type SQSBinding struct {
	SQSConfiguration *SQSConfiguration
	Config           *core.Config
	MessagePump      *core.MessagePump
	HandlerManager   *core.HandlerManager
	done             chan error
}

func (b *SQSBinding) Start(ctx context.Context) {
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

func (b *SQSBinding) Done() <-chan error {
	return b.done
}

func NewSQSBinding(sqsConfig *SQSConfiguration, config *core.Config) *SQSBinding {
	sqs, err := NewSQSService(sqsConfig)
	if err != nil {
		panic(err)
	}
	dispatcher := core.NewHttpDispatcher(config.HandlerURL)
	mp := core.NewMessagePump(sqs, dispatcher, config.RetryCount, config.RetryDelay, config.MaxHandlers)
	hm := core.NewHandlerManager(config.HandlerCommand, config.HandlerArgs, config.HandlerURL, config.StartupDelaySeconds)
	return &SQSBinding{
		SQSConfiguration: sqsConfig,
		Config:           config,
		MessagePump:      mp,
		HandlerManager:   hm,
		done:             make(chan error),
	}
}
