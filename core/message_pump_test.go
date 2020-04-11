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

	transporter := NewMockTransporter(ctrl)
	m := &Message{}
	msgs := make(chan []*Message, 1)
	msgs <- []*Message{m}
	events := make(chan string)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(m).DoAndReturn(func(m *Message) error {
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
	p := NewMessagePump(transporter, d, 0, time.Second, 0)
	p.Start(ctx)

	expectedEvents := []string{"dispatched", "deleted"}
	for e := range events {
		assert.Equal(t, expectedEvents[0], e)
		expectedEvents = expectedEvents[1:]
	}

	cancelFunc()
	close(msgs)
	p.Awaiter.Err()
}

func TestMaxMessageHandlersWithoutBuffering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	events := make(chan string)

	transporter := NewMockTransporter(ctrl)
	m1 := &Message{MessageID: "m1"}
	m2 := &Message{MessageID: "m2"}
	msgs := make(chan []*Message, 2)
	msgs <- []*Message{m1}
	msgs <- []*Message{m2}

	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(gomock.Any()).DoAndReturn(func(m *Message) error {
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
	p := NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)

	expectedOrderOfEvents := []string{"dispatch m1", "delete m1", "dispatch m2", "delete m2"}
	for _, e := range expectedOrderOfEvents {
		assert.Equal(t, e, <-events)
	}

	cancelFunc()
	close(msgs)
	p.Awaiter.Err()
}

func TestMaxMessageHandlersWithBuffering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	events := make(chan string)

	transporter := NewMockTransporter(ctrl)
	m1 := &Message{MessageID: "m1"}
	m2 := &Message{MessageID: "m2"}
	msgs := make(chan []*Message, 1)
	msgs <- []*Message{m1, m2}
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(gomock.Any()).DoAndReturn(func(m *Message) error {
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
	p := NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)

	expectedOrderOfEvents := []string{"dispatch m1", "delete m1", "dispatch m2", "delete m2"}
	for _, e := range expectedOrderOfEvents {
		assert.Equal(t, e, <-events)
	}

	cancelFunc()
	close(msgs)
	p.Awaiter.Err()
}

func TestClosingTransporter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgs := make(chan []*Message)
	transporter := NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()

	d := NewMockDispatcher(ctrl)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	p := NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)

	close(msgs)

	p.Awaiter.Err()
}

func TestCancelationWhileWaitingForDispatcherToReturn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wg := sync.WaitGroup{}
	wg.Add(1)

	transporter := NewMockTransporter(ctrl)
	m1 := &Message{MessageID: "m1"}
	msgs := make(chan []*Message, 1)
	msgs <- []*Message{m1}

	transporter.EXPECT().Messages().Return(msgs).AnyTimes()

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *Message) error {
		wg.Wait()
		return nil
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)

	cancelFunc()
	p.Awaiter.Err()
}

func TestPoisoning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transporter := NewMockTransporter(ctrl)
	m1 := &Message{MessageID: "m1"}
	msgs := make(chan []*Message, 1)
	msgs <- []*Message{m1}
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()

	wg := sync.WaitGroup{}
	wg.Add(1)
	transporter.EXPECT().Poison(m1).DoAndReturn(func(m *Message) error {
		wg.Done()
		return nil
	})

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).DoAndReturn(func(m *Message) error {
		return errors.New("doh")
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)
	wg.Wait()

	cancelFunc()
	close(msgs)
	p.Awaiter.Err()
}
