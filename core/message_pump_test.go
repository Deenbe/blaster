package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	events := make(chan string)
	q.EXPECT().Read().Return([]*Message{m}, nil)
	q.EXPECT().Read().Return([]*Message{}, nil).AnyTimes()
	q.EXPECT().Delete(m).DoAndReturn(func(m *Message) error {
		events <- "deleted"
		close(events)
		return nil
	})

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(m).Do(func(m *Message) error {
		events <- "dispatched"
		return nil
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := NewMessagePump(q, d, 0, time.Second, 0)
	p.Start(ctx)

	expectedEvents := []string{"dispatched", "deleted"}
	for e := range events {
		assert.Equal(t, expectedEvents[0], e)
		expectedEvents = expectedEvents[1:]
	}

	cancelFunc()
	<-p.Done
}

func TestMaxMessageHandlersWithoutBuffering(t *testing.T) {
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

func TestMaxMessageHandlersWithBuffering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	events := make(chan string)

	q := NewMockQueueService(ctrl)
	m1 := &Message{MessageID: "m1"}
	m2 := &Message{MessageID: "m2"}
	q.EXPECT().Read().DoAndReturn(func() ([]*Message, error) {
		events <- "read m1 and m2"
		return []*Message{m1, m2}, nil
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

	expectedOrderOfEvents := []string{"read m1 and m2", "dispatch m1", "delete m1", "dispatch m2", "delete m2"}
	for _, e := range expectedOrderOfEvents {
		assert.Equal(t, e, <-events)
	}
}

func TestErrorOnQueueServiceRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	done := make(chan struct{})

	q := NewMockQueueService(ctrl)
	q.EXPECT().Read().Return([]*Message{}, errors.New("doh"))
	q.EXPECT().Read().DoAndReturn(func() ([]*Message, error) {
		done <- struct{}{}
		return []*Message{}, nil
	}).AnyTimes()

	d := NewMockDispatcher(ctrl)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	p := NewMessagePump(q, d, 0, time.Second, 1)
	p.Start(ctx)

	<-done
}

func TestCancelationWhileWaitingForDispatcherToReturn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wg := sync.WaitGroup{}
	wg.Add(1)

	q := NewMockQueueService(ctrl)
	m1 := &Message{MessageID: "m1"}
	q.EXPECT().Read().Return([]*Message{m1}, nil)
	q.EXPECT().Read().Return([]*Message{}, nil).AnyTimes()
	q.EXPECT().Delete(gomock.Any()).Return(nil).AnyTimes()

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *Message) error {
		wg.Wait()
		return nil
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := NewMessagePump(q, d, 0, time.Second, 1)
	p.Start(ctx)

	cancelFunc()
	<-p.Done
}

func TestPoisoning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	q := NewMockQueueService(ctrl)
	m1 := &Message{MessageID: "m1"}
	q.EXPECT().Read().Return([]*Message{m1}, nil)
	q.EXPECT().Read().Return([]*Message{}, nil).AnyTimes()

	wg := sync.WaitGroup{}
	wg.Add(1)
	q.EXPECT().Poison(m1).DoAndReturn(func(m *Message) error {
		wg.Done()
		return nil
	})

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).DoAndReturn(func(m *Message) error {
		return errors.New("doh")
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := NewMessagePump(q, d, 0, time.Second, 1)
	p.Start(ctx)
	wg.Wait()

	cancelFunc()
	<-p.Done
}
