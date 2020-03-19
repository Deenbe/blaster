package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type Message struct {
	MessageID  *string                `json:"messageId"`
	Body       *string                `json:"body"`
	Properties map[string]interface{} `json:"properties"`
}

type QueueService interface {
	Read() ([]*Message, error)
	Delete(*Message) error
}

type Dispatcher interface {
	Dispatch(*Message) error
}

type MessagePump struct {
	QueueService QueueService
	Dispatcher   Dispatcher
}

type SQSConfiguration struct {
	QueueName           string
	Region              string
	MaxNumberOfMessages int64
	WaitTime            int64
}

func (p *MessagePump) Start() {
	for {
		msgs, err := p.QueueService.Read()
		if err != nil {
			panic(err)
		}

		for _, msg := range msgs {
			go func() {
				e := p.Dispatcher.Dispatch(msg)
				if e != nil {
					panic(e)
				}
				e = p.QueueService.Delete(msg)
				if e != nil {
					panic(e)
				}
			}()
		}
	}
}

func NewMessagePump(queueReader QueueService, dispatcher Dispatcher) *MessagePump {
	return &MessagePump{queueReader, dispatcher}
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
	rh, ok := message.Properties["receiptHandle"].(string)
	if !ok {
		panic("Unexpected input: SQS message without a receipt handle")
	}

	_, err := s.Client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &s.QueueUrl,
		ReceiptHandle: &rh,
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

type HttpDispatcher struct {
	HandlerUrl string
}

func (d *HttpDispatcher) Dispatch(m *Message) error {
	output, err := json.Marshal(m)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = http.Post(d.HandlerUrl, "application/json", bytes.NewReader(output))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func NewHttpDispatcher(handlerUrl string) *HttpDispatcher {
	return &HttpDispatcher{
		HandlerUrl: handlerUrl,
	}
}
