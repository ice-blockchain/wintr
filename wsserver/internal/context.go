// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
)

func NewCustomCancelContext(reqCtx context.Context, ch <-chan struct{}) context.Context {
	return customCancelContext{Context: reqCtx, ch: ch}
}

func (c customCancelContext) Done() <-chan struct{} {
	return c.ch
}

func (c customCancelContext) Err() error {
	select {
	case <-c.ch:
		return context.Canceled
	default:
		return nil
	}
}
