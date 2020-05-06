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

package kafka

import (
	"blaster/core"
	"blaster/mocks"
	"blaster/utils"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func createTestMessage(id string, offset int64) *core.Message {
	return &core.Message{
		Data: map[string]interface{}{
			DataItemMessage: &sarama.ConsumerMessage{Offset: offset},
		},
	}
}

func TestBuferredDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 2, PreCommitHookPath: "a"}
	m1 := createTestMessage("a", 0)
	m2 := createTestMessage("b", 1)
	wg := sync.WaitGroup{}
	commits := make(chan *sarama.ConsumerMessage, 1)

	wg.Add(1)

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m2).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {
		commits <- m
		close(commits)
	})
	c.Start()

	// Act
	err1 := c.Commit(m1)
	err2 := c.Commit(m2)
	wg.Wait()

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, m2.Data[DataItemMessage], <-commits)

	c.Stop()
	<-c.Awaiter().Done()
}

func TestOutOfOrderCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 2, PreCommitHookPath: "a"}
	m1 := createTestMessage("a", 0)
	m2 := createTestMessage("b", 1)
	wg := sync.WaitGroup{}
	commits := make(chan *sarama.ConsumerMessage, 1)

	wg.Add(1)

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m2).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {
		commits <- m
		close(commits)
	})
	c.Start()

	// Act
	err2 := c.Commit(m2)
	err1 := c.Commit(m1)
	wg.Wait()

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, m2.Data[DataItemMessage], <-commits)

	c.Stop()
	<-c.Awaiter().Done()
}

func TestFailingPreCommitHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 1, PreCommitHookPath: "a"}
	m1 := createTestMessage("a", 0)
	wg := sync.WaitGroup{}
	commits := make(chan *sarama.ConsumerMessage, 1)

	wg.Add(1)

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return errors.New("doh")
	})

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {
		commits <- m
	})
	c.Start()

	// Act
	err1 := c.Commit(m1)
	wg.Wait()

	// Assert
	assert.EqualError(t, err1, "doh")

	c.Stop()
	<-c.Awaiter().Done()

	// Ensure that commit func is not invoked for the message
	close(commits)
	assert.Nil(t, <-commits)
}

func TestTimerBeforeAnyMessageIsSeen(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 1}
	dispatcher := mocks.NewMockDispatcher(ctrl)

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {})
	c.Start()

	// Act
	after <- time.Now()

	// Assert
	c.Stop()
	<-c.Awaiter().Done()
}

func TestTimer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 2, CommitIntervalSeconds: 5, PreCommitHookPath: "a"}
	m1 := createTestMessage("a", 0)
	wg := sync.WaitGroup{}
	commits := make(chan *sarama.ConsumerMessage, 1)

	wg.Add(1)

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return nil
	})

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {
		commits <- m
		close(commits)
	})
	c.Start()
	c.Commit(m1)

	// Act
	after <- tc.Now().Add(time.Second * 6)
	wg.Wait()

	// Assert
	assert.Equal(t, m1.Data[DataItemMessage], <-commits)

	c.Stop()
	<-c.Awaiter().Done()
}

func TestTimerBeforeIdleTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 2, CommitIntervalSeconds: 5}
	m1 := createTestMessage("a", 0)

	dispatcher := mocks.NewMockDispatcher(ctrl)
	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {})
	c.Start()
	c.Commit(m1)

	// Act
	after <- tc.Now().Add(time.Second * 3)

	// Assert
	c.Stop()
	<-c.Awaiter().Done()
}

func TestDispatchFailureOnTimer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{CommitBatchSize: 2, CommitIntervalSeconds: 5, PreCommitHookPath: "a"}
	m1 := createTestMessage("a", 0)
	wg := sync.WaitGroup{}
	commits := make(chan *sarama.ConsumerMessage, 1)

	wg.Add(1)

	dispatcher := mocks.NewMockDispatcher(ctrl)
	dispatcher.EXPECT().Dispatch(m1).DoAndReturn(func(m *core.Message) error {
		wg.Done()
		return errors.New("doh")
	})

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(m *sarama.ConsumerMessage) {
		commits <- m
		close(commits)
	})
	c.Start()
	c.Commit(m1)

	// Act
	after <- tc.Now().Add(time.Second * 6)
	wg.Wait()
	c.Stop()
	<-c.Awaiter().Done()
	close(commits)

	// Assert
	assert.Nil(t, <-commits)
}

func TestCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup
	tc := utils.NewTestClock()
	after := make(chan time.Time)
	config := &Config{}
	dispatcher := mocks.NewMockDispatcher(ctrl)

	c := NewBuferedComitter(config, tc.Now, after, dispatcher, func(_ *sarama.ConsumerMessage) {})
	c.Start()

	// Act
	c.Stop()

	// Assert
	<-c.Awaiter().Done()
}

func TestGetLatestMessageForNil(t *testing.T) {
	// Setup
	m := createTestMessage("a", 0)

	// Act
	a := getLatestMessage(m, nil)
	b := getLatestMessage(nil, m)

	// Assert
	assert.Equal(t, m, a)
	assert.Equal(t, m, b)
}
