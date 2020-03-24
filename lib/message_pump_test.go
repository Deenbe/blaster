package lib

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func TestBasicMessageDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var wg sync.WaitGroup
	wg.Add(1)

	q := NewMockQueueService(ctrl)
	m := new(Message)
	q.EXPECT().Read().Return([]*Message{m}, nil)
	q.EXPECT().Delete(gomock.Any()).Return(nil)

	d := NewMockDispatcher(ctrl)
	d.EXPECT().Dispatch(gomock.Any()).Do(func(m *Message) error { 
		wg.Done() 
		return nil
	})

	p := NewMessagePump(q, d, 0, time.Second)
	p.pumpMain()

	wg.Wait()
}
