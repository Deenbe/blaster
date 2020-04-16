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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDefaultMessagePumpRunnerExit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup a mock Transporter
	messages := make(chan []*core.Message)
	defer close(messages)
	transporter := mocks.NewMockTransporter(ctrl)
	transporter.EXPECT().Messages().Return(messages)

	// Test config
	// Use sleep 10 as our handler command as we
	// don't receive any messages in this test.
	config := core.Config{
		HandlerCommand:      "sleep",
		HandlerArgs:         []string{"20"},
		StartupDelaySeconds: 1,
	}

	runner := core.NewDefaultMessagePumpRunner()
	ctx, cancelFunc := context.WithCancel(context.Background())
	awaiter := runner.Run(ctx, transporter, config)

	// Signal cancellation
	cancelFunc()
	// Simulate an empty receive to unblock message pump
	messages <- []*core.Message{}

	err := awaiter.Err()
	assert.Error(t, err)
}
