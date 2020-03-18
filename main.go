package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type RequestMessage struct {
	MessageID              *string                               `json:"messageId"`
	Body                   *string                               `json:"body"`
	Attributes             map[string]*string                    `json:"attributes"`
	MessageAttributes      map[string]*sqs.MessageAttributeValue `json:"messageAttributes"`
	MD5OfBody              *string                               `json:"md5OfBody"`
	MD5OfMessageAttributes *string                               `json:"md5OfMessageAttributes"`
	ReceiptHandle          *string                               `json:"receiptHandle"`
}

func dispatch(svc *sqs.SQS, url *string, m *sqs.Message) {
	rm := &RequestMessage{
		MessageID:              m.MessageId,
		Body:                   m.Body,
		Attributes:             m.Attributes,
		MessageAttributes:      m.MessageAttributes,
		MD5OfBody:              m.MD5OfBody,
		MD5OfMessageAttributes: m.MD5OfMessageAttributes,
		ReceiptHandle:          m.ReceiptHandle,
	}

	output, err := json.Marshal(rm)
	if err != nil {
		panic(err)
	}

	_, err = http.Post("http://localhost:9000/", "application/json", bytes.NewReader(output))
	if err != nil {
		panic(err)
	}

	_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      url,
		ReceiptHandle: m.ReceiptHandle,
	})

	if err != nil {
		panic(err)
	}
}

func main() {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-southeast-2"),
	}))

	svc := sqs.New(s)
	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String("fc-poc"),
	})
	if err != nil {
		panic(err)
	}

	for {
		result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            urlResult.QueueUrl,
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(10),
		})

		if err != nil {
			panic(err)
		}

		fmt.Printf("Received %d \n", len(result.Messages))
		for _, m := range result.Messages {
			go dispatch(svc, urlResult.QueueUrl, m)
		}
	}
}
