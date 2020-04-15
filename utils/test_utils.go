// +build integration

package utils

import (
	"blaster/core"
	"context"
)

func AwaiterForCancelContext(ctx context.Context) *core.Awaiter {
	awaiter, awaitNotifier := core.NewAwaiter()

	go func() {
		<-ctx.Done()
		awaitNotifier.Notify(ctx.Err())
	}()

	return awaiter
}
