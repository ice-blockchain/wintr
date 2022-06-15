// SPDX-License-Identifier: BUSL-1.1

package time

import (
	"strconv"
	stdlibtime "time"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/ice-blockchain/wintr/log"
)

func Now() *Time {
	now := stdlibtime.Now().UTC()

	return &Time{
		Time: &now,
	}
}

func New(time stdlibtime.Time) *Time {
	return &Time{
		Time: &time,
	}
}

func (t *Time) DecodeMsgpack(dec *msgpack.Decoder) error {
	nanoSecs, err := dec.DecodeUint64()
	if err != nil {
		return errors.Wrap(err, "failed to Time.DecodeMsgpack.DecodeBytes")
	}
	t.Time = new(stdlibtime.Time)
	*t.Time = stdlibtime.Unix(0, int64(nanoSecs)).UTC()

	return nil
}

func (t *Time) EncodeMsgpack(enc *msgpack.Encoder) error {
	var nanos uint64
	if t.Location() != stdlibtime.UTC {
		nanos = uint64(t.UTC().UnixNano())
	} else {
		nanos = uint64(t.UnixNano())
	}

	return errors.Wrap(enc.EncodeUint64(nanos), "failed to EncodeUint64")
}

func (t *Time) MarshalJSON() ([]byte, error) {
	if t.Location() != stdlibtime.UTC {
		*t.Time = t.Time.UTC()
	}

	//nolint:wrapcheck // We're just proxying it.
	return t.Time.MarshalJSON()
}

func (t *Time) UnmarshalJSON(bytes []byte) (err error) {
	t.unmarshallUint64(bytes)
	if t.Time != nil {
		return nil
	}

	return t.unmarshallString(bytes)
}

func (t *Time) unmarshallUint64(data []byte) {
	for _, b := range data {
		if b < 48 || b > 57 {
			return
		}
	}
	millisOrNanos, err := strconv.Atoi(string(data))
	log.Panic(err)
	t.Time = new(stdlibtime.Time)
	//nolint:gomnd // There's no magic here, there are 13 digits in a millisecond based timestamp.
	if len(data) == 13 {
		*t.Time = stdlibtime.UnixMilli(int64(millisOrNanos)).UTC()
	} else {
		*t.Time = stdlibtime.Unix(0, int64(millisOrNanos)).UTC()
	}
}

func (t *Time) unmarshallString(bytes []byte) error {
	data := string(bytes)
	if data == "null" || data == `""` || data == "" {
		return nil
	}
	time, err := stdlibtime.Parse(`"`+stdlibtime.RFC3339Nano+`"`, data)
	if err != nil {
		return errors.Wrapf(err, "invalid time format: %v", data)
	}
	t.Time = new(stdlibtime.Time)
	*t.Time = time.UTC()

	return nil
}
