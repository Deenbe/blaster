// +build integration

package sqs_test

import (
	"blaster/core"
	"blaster/mocks"
	binder "blaster/sqs"
	"blaster/utils"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testQueue struct {
	name   string
	url    string
	sqssvc *sqs.SQS
}

func setupQueue(t *testing.T) *testQueue {
	buf := make([]byte, 16)
	count, err := rand.Read(buf)
	assert.NoError(t, err)
	if count != len(buf) {
		t.Fatal(errors.New("insufficient data to generate queue name"))
	}

	name := fmt.Sprintf("blaster-integration-test-%s", hex.EncodeToString(buf))
	sess := session.Must(session.NewSessionWithOptions(session.Options{}))
	sqssvc := sqs.New(sess)
	result, err := sqssvc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(name),
	})
	assert.NoError(t, err)

	return &testQueue{
		name:   name,
		url:    *result.QueueUrl,
		sqssvc: sqssvc,
	}
}

func (q *testQueue) Delete(t *testing.T) {
	_, err := q.sqssvc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: &q.url,
	})

	assert.NoError(t, err)
}

func TestSQS(t *testing.T) {
	q := setupQueue(t)
	defer q.Delete(t)
	t.Run("EndToEnd", func(t *testing.T) { EndToEnd(t, q) })
}

func EndToEnd(t *testing.T, q *testQueue) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancelFunc := context.WithCancel(context.Background())
	awaiter := utils.AwaiterForCancelContext(ctx)
	messages := make(chan []*core.Message)

	// Setup MessagePumpRunner to read a message off the transport and delete it.
	runner := mocks.NewMockMessagePumpRunner(ctrl)
	runner.
		EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, transporter core.Transporter, config core.Config) *core.Awaiter {
			go func() {
				batch := <-transporter.Messages()
				for _, m := range batch {
					transporter.Delete(m)
				}
				messages <- batch
				close(messages)
			}()
			return awaiter
		})

	binder, err := (&binder.SQSBinderBuilder{}).Build(runner, &core.Config{}, binder.SQSConfiguration{
		QueueName:           q.name,
		MaxNumberOfMessages: 1,
		WaitTime:            0,
	})

	assert.NoError(t, err)
	binder.Start(ctx)

	_, err = q.sqssvc.SendMessage(&sqs.SendMessageInput{QueueUrl: &q.url, MessageBody: aws.String("hey")})
	assert.NoError(t, err)

	received := <-messages

	assert.Len(t, received, 1)
	assert.Equal(t, "hey", received[0].Body)

	// Ensure the message is deleted
	attribs, err := q.sqssvc.GetQueueAttributes(&sqs.GetQueueAttributesInput{QueueUrl: &q.url, AttributeNames: []*string{aws.String(sqs.QueueAttributeNameApproximateNumberOfMessages)}})
	assert.NoError(t, err)
	assert.Equal(t, "0", *attribs.Attributes[sqs.QueueAttributeNameApproximateNumberOfMessages])

	cancelFunc()
	err = binder.Awaiter().Err()
	assert.EqualError(t, err, "sqs binder exited")
}
