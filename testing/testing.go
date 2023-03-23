// SPDX-License-Identifier: ice License 1.0

package testing

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-json"
	"github.com/goccy/go-reflect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func GIVEN(_ string, logic func()) {
	logic()
}

func WHEN(_ string, logic func()) {
	logic()
}

func THEN(logic func()) {
	logic()
}

func IT(_ string, logic func()) {
	logic()
}

func AND(_ string, logic func()) {
	logic()
}

func SETUP(_ string, logic func()) {
	logic()
}

func AssertSymmetricMarshallingUnmarshalling[OBJ any](tb testing.TB, expectedUnmarshalling *OBJ, expectedMarshalling string, expectedEmptyMarshallingArg ...string) { //nolint:lll // .
	tb.Helper()
	expectedMarshallingCompactedBuffer := new(bytes.Buffer)
	require.NoError(tb, json.Compact(expectedMarshallingCompactedBuffer, []byte(expectedMarshalling)))
	// Marshalling.
	expectedEmptyMarshalling := "{}"
	if len(expectedEmptyMarshallingArg) == 1 {
		expectedEmptyMarshalling = expectedEmptyMarshallingArg[0]
	}
	expectedEmptyMarshallingCompactedBuffer := new(bytes.Buffer)
	require.NoError(tb, json.Compact(expectedEmptyMarshallingCompactedBuffer, []byte(expectedEmptyMarshalling)))
	assert.Equal(tb, expectedEmptyMarshallingCompactedBuffer.String(), MustMarshal(tb, new(OBJ)))
	assert.Equal(tb, expectedMarshallingCompactedBuffer.String(), MustMarshal(tb, expectedUnmarshalling))
	// Unmarshalling.
	assert.EqualValues(tb, new(OBJ), MustUnmarshal[OBJ](tb, "{}"))
	zeroValueIgnoredFields(expectedUnmarshalling)
	assert.EqualValues(tb, expectedUnmarshalling, MustUnmarshal[OBJ](tb, expectedMarshalling))
}

func zeroValueIgnoredFields(val any) {
	vType := reflect.TypeOf(val).Elem()
	vValue := reflect.ValueOf(val).Elem()
	for ix := 0; ix < vType.NumField(); ix++ {
		if vType.Field(ix).PkgPath != "" {
			continue
		}
		if jsonTag := vType.Field(ix).Tag.Get("json"); jsonTag == "-" {
			vValue.Field(ix).Set(reflect.Zero(vType.Field(ix).Type))
		}
		if vValue.Field(ix).Kind() == reflect.Struct {
			zeroValueIgnoredFields(vValue.Field(ix).Addr().Interface())
		}
		if vValue.Field(ix).Kind() == reflect.Ptr {
			if vValue.Field(ix).Elem().Kind() == reflect.Struct {
				zeroValueIgnoredFields(vValue.Field(ix).Interface())
			}
		}
	}
}

func MustMarshal(tb testing.TB, val any) string {
	tb.Helper()
	valueBytes, err := json.MarshalContext(context.Background(), val)
	require.NoError(tb, err)

	return string(valueBytes)
}

func MustUnmarshal[T any](tb testing.TB, val string) *T {
	tb.Helper()
	tt := new(T)
	require.NoError(tb, json.UnmarshalContext(context.Background(), []byte(val), tt))

	return tt
}
