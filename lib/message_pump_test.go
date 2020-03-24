package lib

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBasicMessageDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	q := NewMockQueueService(ctrl)
	m := new(Message)
	q.EXPECT().Read().Return([]*Message{m}, nil)
	q.EXPECT().Delete(m).Return(nil)

	d := NewMockDispatcher(ctrl)
	observe := make(chan *Message)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *Message) error { 
		observe <- m
		return nil
	})

	p := NewMessagePump(q, d, 0, time.Second)
	p.pumpMain()

	assert.Equal(t, m, <-observe)
}
