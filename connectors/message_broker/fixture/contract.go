// SPDX-License-Identifier: BUSL-1.1

package fixture

import "sync"

// Public API.

type (
	RawMessage struct {
		key, value, topic string
	}

	TestMessageBroker struct{}
)

// Private API.

var (
	// nolint:gochecknoglobals // It`s a stateless singleton
	globalMessageSource []RawMessage
	// nolint:gochecknoglobals // It`s a stateless singleton
	mx = new(sync.RWMutex)
)
