package lib

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBasicMessageDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	q := NewMockQueueService(ctrl)
	m := &Message{}
	q.EXPECT().Read().Return([]*Message{m}, nil)
	q.EXPECT().Read().Return([]*Message{}, nil).AnyTimes()
	q.EXPECT().Delete(m).Return(nil)

	d := NewMockDispatcher(ctrl)
	observe := make(chan *Message)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *Message) error {
		observe <- m
		return nil
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := NewMessagePump(q, d, 0, time.Second, 0)
	p.Start(ctx)

	assert.Equal(t, m, <-observe)

	cancelFunc()
	<-p.Done
}

func TestMaxMessageHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	events := make(chan string)

	q := NewMockQueueService(ctrl)
	m1 := &Message{MessageID: "m1"}
	m2 := &Message{MessageID: "m2"}
	q.EXPECT().Read().DoAndReturn(func() ([]*Message, error) {
		events <- "read m1"
		return []*Message{m1}, nil
	})
	q.EXPECT().Read().DoAndReturn(func() ([]*Message, error) {
		events <- "read m2"
		return []*Message{m2}, nil
	})
	q.EXPECT().Read().Return([]*Message{}, nil).AnyTimes()
	q.EXPECT().Delete(gomock.Any()).DoAndReturn(func(m *Message) error {
		events <- fmt.Sprintf("delete %s", m.MessageID)
		return nil
	}).AnyTimes()

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *Message) error {
		events <- fmt.Sprintf("dispatch %s", m.MessageID)
		return nil
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	p := NewMessagePump(q, d, 0, time.Second, 1)
	p.Start(ctx)

	expectedOrderOfEvents := []string{"read m1", "dispatch m1", "delete m1", "read m2", "dispatch m2", "delete m2"}
	for _, e := range expectedOrderOfEvents {
		assert.Equal(t, e, <-events)
	}
}
