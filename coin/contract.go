// SPDX-License-Identifier: BUSL-1.1

package coin

import (
	"encoding/json"

	"cosmossdk.io/math"
	"github.com/vmihailenco/msgpack/v5"
)

// Public API.

// Denomination is the factor between `ice` and its smallest subunit called `ice flake`.
const Denomination = 1e9

type (
	ICE      string
	ICEFlake struct {
		math.Uint
	}
	Coin struct {
		// Amount is anything between `[0,2^256)`, where `1 ice = 1E9 * iceflakes`.
		Amount *ICEFlake `json:"amount,omitempty" example:"115792089237316195423570985008687907853269984665640564039457584007913129639935"`
	}
)

// Private API.

// See Denomination.
const e9 = 9

var (
	_ msgpack.CustomEncoder = (*ICEFlake)(nil)
	_ msgpack.CustomDecoder = (*ICEFlake)(nil)
	_ json.Unmarshaler      = (*ICEFlake)(nil)
	_ json.Marshaler        = (*ICEFlake)(nil)
	_ json.Unmarshaler      = (*ICE)(nil)
	_ json.Marshaler        = (*ICE)(nil)
	//nolint:gochecknoglobals // Its goroutine safe.
	denomination = math.NewUint(Denomination)
)
