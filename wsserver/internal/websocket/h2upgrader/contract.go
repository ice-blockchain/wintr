package h2upgrader

import "github.com/gobwas/httphead"

// Implements PFC 8441.
//

type (
	H2Upgrader struct {
		Protocol  func(string) bool
		Extension func(httphead.Option) bool
		Negotiate func(httphead.Option) (httphead.Option, error)
	}
)

const (
	headerSecVersionCanonical    = "Sec-Websocket-Version"
	headerSecProtocolCanonical   = "Sec-Websocket-Protocol"
	headerSecExtensionsCanonical = "Sec-Websocket-Extensions"
)
