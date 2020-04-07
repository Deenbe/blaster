package sqs

import (
	"blaster/core"
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	messages      chan []*core.Message
	Awaiter       *core.Awaiter
	awaitNotifier *core.AwaitNotifier
	logFields     log.Fields
}

func (t *SQSTransporter) Messages() <-chan []*core.Message {
	return t.messages
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

func (t *SQSTransporter) Start(ctx context.Context) {
	go func() {
		for {
			batch, err := t.read()
			if err != nil {
				log.WithFields(t.logFields).WithFields(log.Fields{"err": err}).Info("error receiving messages from sqs")
				batch = core.EmptyMessageSet
			}

			select {
			case t.messages <- batch:
			case <-ctx.Done():
				close(t.messages)
				t.awaitNotifier.Notify(ctx.Err())
				return
			}
		}
	}()
}

func (t *SQSTransporter) read() ([]*core.Message, error) {
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

	logFields := log.Fields{"module": "sqs_transporter"}
	awaiter, awaitNotifier := core.NewAwaiter()
	return &SQSTransporter{
		Configuration: configuration,
		Client:        svc,
		QueueUrl:      *urlResult.QueueUrl,
		Awaiter:       awaiter,
		awaitNotifier: awaitNotifier,
		logFields:     logFields,
		messages:      make(chan []*core.Message),
	}, nil
}

type SQSBinder struct {
	SQSConfiguration *SQSConfiguration
	Config           *core.Config
	Transporter      *SQSTransporter
	MessagePump      *core.MessagePump
	HandlerManager   *core.HandlerManager
	logFields        log.Fields
	awaiter          *core.Awaiter
	awaitNotifier    *core.AwaitNotifier
}

func (b *SQSBinder) Start(ctx context.Context) {
	go func() {
		b.HandlerManager.Start(ctx)
		b.MessagePump.Start(ctx)
		b.Transporter.Start(ctx)

		select {
		case <-b.MessagePump.Awaiter.Done():
		case <-b.HandlerManager.Awaiter.Done():
		case <-b.Transporter.Awaiter.Done():
		}

		err := b.MessagePump.Awaiter.Err()
		log.WithFields(b.logFields).WithField("err", err).Info("message pump exited")
		err = b.HandlerManager.Awaiter.Err()
		log.WithFields(b.logFields).WithField("err", err).Info("handler manager exited")
		err = b.Transporter.Awaiter.Err()
		log.WithFields(b.logFields).WithField("err", err).Info("sqs transporter exited")

		b.awaitNotifier.Notify(nil)
	}()
}

func (b *SQSBinder) Awaiter() *core.Awaiter {
	return b.awaiter
}

func NewSQSBinder(sqsConfig *SQSConfiguration, config *core.Config) *SQSBinder {
	transporter, err := NewSQSTransporter(sqsConfig)
	if err != nil {
		panic(err)
	}
	dispatcher := core.NewHttpDispatcher(config.HandlerURL)
	mp := core.NewMessagePump(transporter, dispatcher, config.RetryCount, config.RetryDelay, config.MaxHandlers)
	hm := core.NewHandlerManager(config.HandlerCommand, config.HandlerArgs, config.HandlerURL, config.StartupDelaySeconds)
	awaiter, awaitNotifier := core.NewAwaiter()
	return &SQSBinder{
		SQSConfiguration: sqsConfig,
		Config:           config,
		MessagePump:      mp,
		HandlerManager:   hm,
		Transporter:      transporter,
		logFields:        log.Fields{"module": "sqs_binder"},
		awaiter:          awaiter,
		awaitNotifier:    awaitNotifier,
	}
}
