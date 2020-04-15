/*
Copyright © 2020 Blaster Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	s := session.Must(session.NewSessionWithOptions(session.Options{}))

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
	runner           core.MessagePumpRunner
	logFields        log.Fields
	awaiter          *core.Awaiter
	awaitNotifier    *core.AwaitNotifier
}

func (b *SQSBinder) Start(ctx context.Context) {
	go func() {
		runner := b.runner.Run(ctx, b.Transporter, *b.Config)
		b.Transporter.Start(ctx)

		select {
		case <-runner.Done():
		case <-b.Transporter.Awaiter.Done():
		}

		err := runner.Err()
		log.WithFields(b.logFields).WithField("err", err).Info("message pump runner exited")
		err = b.Transporter.Awaiter.Err()
		log.WithFields(b.logFields).WithField("err", err).Info("sqs transporter exited")

		b.awaitNotifier.Notify(errors.New("sqs binder exited"))
	}()
}

func (b *SQSBinder) Awaiter() *core.Awaiter {
	return b.awaiter
}

type SQSBinderBuilder struct {
}

func (b *SQSBinderBuilder) Build(runner core.MessagePumpRunner, config *core.Config, options interface{}) (core.BrokerBinder, error) {
	sqsConfig := options.(SQSConfiguration)
	transporter, err := NewSQSTransporter(&sqsConfig)
	if err != nil {
		return nil, err
	}
	awaiter, awaitNotifier := core.NewAwaiter()
	return &SQSBinder{
		SQSConfiguration: &sqsConfig,
		Config:           config,
		Transporter:      transporter,
		logFields:        log.Fields{"module": "sqs_binder"},
		runner:           runner,
		awaiter:          awaiter,
		awaitNotifier:    awaitNotifier,
	}, nil
}
