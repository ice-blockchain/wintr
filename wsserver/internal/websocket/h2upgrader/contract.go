// SPDX-License-Identifier: ice License 1.0

package h2upgrader

import (
	"github.com/gobwas/httphead"
	"github.com/pkg/errors"
)

// Implements PFC 8441.
type (
	H2Upgrader struct {
		Protocol  func(string) bool
		Extension func(httphead.Option) bool
		Negotiate func(httphead.Option) (httphead.Option, error)
	}
)

var (
	ErrBadPath     = errors.New(":scheme and :path is required")
	ErrBadProtocol = errors.New(":protocol must be websocket")
)

const (
	headerSecVersionCanonical    = "Sec-Websocket-Version"
	headerSecProtocolCanonical   = "Sec-Websocket-Protocol"
	headerSecExtensionsCanonical = "Sec-Websocket-Extensions"
)
