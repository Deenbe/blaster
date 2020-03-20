package main

import (
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

func (s *SQSService) Read() ([]*Message, error) {
	msgs, err := s.Client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            &s.QueueUrl,
		MaxNumberOfMessages: &s.Configuration.MaxNumberOfMessages,
		WaitTimeSeconds:     &s.Configuration.WaitTime,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := make([]*Message, len(msgs.Messages))

	for i, msg := range msgs.Messages {
		result[i] = &Message{
			MessageID: msg.MessageId,
			Body:      msg.Body,
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

func (s *SQSService) Delete(message *Message) error {
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
