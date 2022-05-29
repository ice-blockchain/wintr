// SPDX-License-Identifier: BUSL-1.1

package time

import (
	"encoding/json"
	stdlibtime "time"

	"github.com/vmihailenco/msgpack/v5"
)

// Public API.

type (
	Time struct {
		*stdlibtime.Time
	}
)

// Private API.

var (
	_ msgpack.CustomEncoder = (*Time)(nil)
	_ msgpack.CustomDecoder = (*Time)(nil)
	_ json.Unmarshaler      = (*Time)(nil)
	_ json.Marshaler        = (*Time)(nil)
)
