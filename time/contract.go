// SPDX-License-Identifier: ice License 1.0

package time

import (
	"database/sql"
	stdlibtime "time"

	"github.com/goccy/go-json"
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
	_ msgpack.CustomEncoder                        = (*Time)(nil)
	_ msgpack.CustomDecoder                        = (*Time)(nil)
	_ json.UnmarshalerContext                      = (*Time)(nil)
	_ json.MarshalerContext                        = (*Time)(nil)
	_ sql.Scanner                                  = (*Time)(nil)
	_ interface{ MarshalBinary() ([]byte, error) } = (*Time)(nil)
	_ interface{ MarshalText() ([]byte, error) }   = (*Time)(nil)
	_ interface{ UnmarshalBinary([]byte) error }   = (*Time)(nil)
	_ interface{ UnmarshalText([]byte) error }     = (*Time)(nil)
)
