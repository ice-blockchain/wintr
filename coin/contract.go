// SPDX-License-Identifier: ice License 1.0

package coin

import (
	"database/sql"

	"cosmossdk.io/math"
	"github.com/goccy/go-json"
	"github.com/vmihailenco/msgpack/v5"
)

// Public API.

const (
	// Denomination is the factor between `ice` and its smallest subunit called `ice flake`.
	Denomination = 1e9
)

type (
	ICE      string
	ICEFlake struct {
		math.Uint
	}
	AmountWords struct {
		AmountWord0 uint64 `json:"-" swaggerignore:"true"`
		AmountWord1 uint64 `json:"-" swaggerignore:"true"`
		AmountWord2 uint64 `json:"-" swaggerignore:"true"`
		AmountWord3 uint64 `json:"-" swaggerignore:"true"`
	}
	Coin struct {
		// Amount is anything between `[0,2^256)`, where `1 ice = 1E9 * iceflakes`.
		// Use ONLY Coin.setAmount to change the Amount.
		Amount *ICEFlake `json:"amount,omitempty" example:"115792089237316195423570985008687907853269984665640564039457584007913129639935"`
		// AmountWords is the uint256 bits representation of the Amount. It's formed out of 4 uint64 math/big.Word`s.
		// Use Coin.setAmount to synchronize Amount with AmountWords.
		AmountWords
	}
)

// Private API.

const (
	// See Denomination.
	e9      = 9
	e9Zeros = "000000000"
	zero    = "0.0"
)

var (
	_ msgpack.CustomEncoder   = (*ICEFlake)(nil)
	_ msgpack.CustomDecoder   = (*ICEFlake)(nil)
	_ msgpack.CustomDecoder   = (*ICE)(nil)
	_ json.UnmarshalerContext = (*ICEFlake)(nil)
	_ json.MarshalerContext   = (*ICEFlake)(nil)
	_ json.UnmarshalerContext = (*ICE)(nil)
	_ json.MarshalerContext   = (*ICE)(nil)
	_ sql.Scanner             = (*ICE)(nil)
	_ sql.Scanner             = (*ICEFlake)(nil)
	//nolint:gochecknoglobals // Its goroutine safe.
	denomination = math.NewUint(Denomination)
)
