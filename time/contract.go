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
	_ msgpack.CustomEncoder   = (*Time)(nil)
	_ msgpack.CustomDecoder   = (*Time)(nil)
	_ json.UnmarshalerContext = (*Time)(nil)
	_ json.MarshalerContext   = (*Time)(nil)
	_ sql.Scanner             = (*Time)(nil)
)
