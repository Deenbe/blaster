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

package core_test

import (
	"blaster/core"
	"blaster/mocks"
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

	// Setup
	m := &core.Message{}
	msgs := make(chan []*core.Message, 1)
	events := make(chan string)

	msgs <- []*core.Message{m}

	// Setup Transporter and Dispatcher so that the methods
	// invoked by MessagePump are recorded in order.
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(m).DoAndReturn(func(m *core.Message) error {
		events <- "deleted"
		close(events)
		return nil
	})

	d := mocks.NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(m).Do(func(m *core.Message) error {
		events <- "dispatched"
		return nil
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, d, 0, time.Second, 0)

	// Act
	p.Start(ctx)

	// Assert
	// Ensure method calls are observed in the expected order
	expectedEvents := []string{"dispatched", "deleted"}
	for e := range events {
		assert.Equal(t, expectedEvents[0], e)
		expectedEvents = expectedEvents[1:]
	}

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

// Messages read from the Transporter are dispatched in one or more
// MessagePump cycles. Only a subset of messages may be delivered if
// MessagePump has reached the MaxWorkers limit. In this case MessagePump
// should not read further messages
// from Transporter until all messages are dispatched.
// Following tests are to ensure messages are dispatched correctly
// under all conditions.
func TestMaxMessageHandlersWithoutBuffering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	m2 := &core.Message{MessageID: "m2"}
	msgs := make(chan []*core.Message, 2)
	events := make(chan string)

	msgs <- []*core.Message{m1}
	msgs <- []*core.Message{m2}

	// Setup Transporter and Dispatcher so that method invocations
	// are recorded in the order they are made.
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(gomock.Any()).DoAndReturn(func(m *core.Message) error {
		events <- fmt.Sprintf("delete %s", m.MessageID)
		return nil
	}).AnyTimes()

	d := mocks.NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *core.Message) error {
		events <- fmt.Sprintf("dispatch %s", m.MessageID)
		return nil
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	p := core.NewMessagePump(transporter, d, 0, time.Second, 1)

	// Act
	p.Start(ctx)

	// Assert
	// Ensure method calls are observed in the expected order
	expectedOrderOfEvents := []string{"dispatch m1", "delete m1", "dispatch m2", "delete m2"}
	for _, e := range expectedOrderOfEvents {
		assert.Equal(t, e, <-events)
	}

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

func TestMaxMessageHandlersWithBuffering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	m2 := &core.Message{MessageID: "m2"}
	msgs := make(chan []*core.Message, 1)
	events := make(chan string)

	msgs <- []*core.Message{m1, m2}

	// Setup Transporter and Dispatcher so that method invocations
	// are recorded in the order they are made.
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(gomock.Any()).DoAndReturn(func(m *core.Message) error {
		events <- fmt.Sprintf("delete %s", m.MessageID)
		return nil
	}).AnyTimes()

	d := mocks.NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *core.Message) error {
		events <- fmt.Sprintf("dispatch %s", m.MessageID)
		return nil
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	p := core.NewMessagePump(transporter, d, 0, time.Second, 1)

	// Act
	p.Start(ctx)

	// Assert
	// Ensure method calls are observed in the expected order
	expectedOrderOfEvents := []string{"dispatch m1", "delete m1", "dispatch m2", "delete m2"}
	for _, e := range expectedOrderOfEvents {
		assert.Equal(t, e, <-events)
	}

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

// Closing the Transporter should unblock MessagePump and eventually
// exit.
func TestClosingTransporter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	msgs := make(chan []*core.Message)

	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()

	d := mocks.NewMockDispatcher(ctrl)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	p := core.NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)

	// Act
	close(msgs)

	// Assert
	<-p.Awaiter().Done()
}

func TestCancelationWhileWaitingForDispatcherToReturn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	wg := sync.WaitGroup{}
	m1 := &core.Message{MessageID: "m1"}
	msgs := make(chan []*core.Message, 1)

	msgs <- []*core.Message{m1}
	wg.Add(1)

	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()

	d := mocks.NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *core.Message) error {
		wg.Wait()
		return nil
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, d, 0, time.Second, 1)
	p.Start(ctx)

	// Act
	cancelFunc()

	// Assert
	<-p.Awaiter().Done()
}

func TestPoisoning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	msgs := make(chan []*core.Message, 1)
	wg := sync.WaitGroup{}

	msgs <- []*core.Message{m1}
	wg.Add(1)

	// Setup the Transporter so that test can only
	// resume when Poison method is invoked.
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Poison(m1).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	d := mocks.NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).DoAndReturn(func(m *core.Message) error {
		return errors.New("doh")
	}).AnyTimes()

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, d, 0, time.Second, 1)

	// Act
	p.Start(ctx)

	// Assert
	wg.Wait()

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

func TestRetryPolicyOnDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	msgs := make(chan []*core.Message, 1)
	wg := sync.WaitGroup{}

	msgs <- []*core.Message{m1}
	wg.Add(1)

	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(m1).Return(nil)

	// Setup the dispatcher so that, it only succeeds during the
	// second attempt to dispatch the message.
	// Test will only resume if wg is signaled.
	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).Return(errors.New("doh"))
	dispatcher.EXPECT().Dispatch(m1).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, dispatcher, 1, time.Nanosecond, 0)

	// Act
	p.Start(ctx)

	// Assert
	wg.Wait()

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

// Ensure that failure while attempting to handle a poison
// message does not compromise the liveness of MessagePump.
func TestFailureDuringPoisonMessageHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	m2 := &core.Message{MessageID: "m2"}
	msgs := make(chan []*core.Message, 2)
	wg := sync.WaitGroup{}

	msgs <- []*core.Message{m1}
	msgs <- []*core.Message{m2}
	wg.Add(1)

	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Poison(m1).Return(errors.New("doh"))
	transporter.EXPECT().Delete(m2).Return(nil)

	// Setup the dispatcher so that it fails to dispatch m1 but
	// succeeds for m2. Successful m2 will unblock the test execution.
	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).Return(errors.New("doh"))
	dispatcher.EXPECT().Dispatch(m2).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, dispatcher, 0, 0, 1)

	// Act
	p.Start(ctx)

	// Assert
	wg.Wait()

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

func TestRetryPolicyOnDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	msgs := make(chan []*core.Message, 1)
	wg := sync.WaitGroup{}

	msgs <- []*core.Message{m1}
	wg.Add(1)

	// Setup the transporter so that, it only succeeds during the
	// second attempt to delete the message.
	// Test will only resume if wg is signaled.
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(m1).Return(errors.New("doh"))
	transporter.EXPECT().Delete(m1).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).Return(nil)

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, dispatcher, 1, time.Nanosecond, 0)

	// Act
	p.Start(ctx)

	// Assert
	wg.Wait()

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}

// Ensure that failure while attempting to handle a poison
// message does not compromise the liveness of MessagePump.
func TestFailureToDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	m1 := &core.Message{MessageID: "m1"}
	m2 := &core.Message{MessageID: "m2"}
	msgs := make(chan []*core.Message, 2)
	wg := sync.WaitGroup{}

	msgs <- []*core.Message{m1}
	msgs <- []*core.Message{m2}
	wg.Add(1)

	// Setup the transporter so that, test is resumed when m2
	// is deleted.
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(msgs).AnyTimes()
	transporter.EXPECT().Delete(m1).Return(errors.New("doh"))
	transporter.EXPECT().Delete(m2).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).Return(nil)
	dispatcher.EXPECT().Dispatch(m2).Return(nil)

	ctx, cancelFunc := context.WithCancel(context.Background())
	p := core.NewMessagePump(transporter, dispatcher, 0, 0, 1)

	// Act
	p.Start(ctx)

	// Assert
	wg.Wait()

	cancelFunc()
	close(msgs)
	<-p.Awaiter().Done()
}
