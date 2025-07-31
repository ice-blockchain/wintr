// SPDX-License-Identifier: ice License 1.0

package sms

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/sms/fixture"
	"github.com/ice-blockchain/wintr/terror"
	"github.com/ice-blockchain/wintr/time"
)

const (
	testApplicationYAMLKey = "self"
)

// .
var (
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	client Client
)

func TestMain(m *testing.M) {
	client = New(testApplicationYAMLKey)
	os.Exit(m.Run()) //nolint:revive // .
}

func TestClientVerifyPhoneNumber(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()

	require.ErrorIs(t, client.VerifyPhoneNumber(ctx, "bogus"), ErrInvalidPhoneNumber)
	err := client.VerifyPhoneNumber(ctx, "+40721 555 524")
	require.ErrorIs(t, err, ErrInvalidPhoneNumberFormat)
	tErr := terror.As(err)
	require.Error(t, tErr)
	require.EqualValues(t, map[string]any{"phoneNumber": "+40721555524"}, tErr.Data)
}

func TestClientSend(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	fixture.TestingPhoneNumber()
	p1 := &Parcel{
		ToNumber: fixture.TestingPhoneNumber(),
		Message:  "123456",
	}
	require.NoError(t, fixture.ClearSMSQueue(p1.ToNumber))
	require.NoError(t, client.Send(ctx, p1))
	fixture.AssertSMSCode(ctx, t, p1.ToNumber, p1.Message, func(body string) string {
		return body
	})
	p1 = &Parcel{
		SendAt:   time.Now(),
		ToNumber: fixture.TestingPhoneNumber(),
		Message:  "123456",
	}
	require.NoError(t, fixture.ClearSMSQueue(p1.ToNumber))
	require.ErrorIs(t, client.Send(ctx, p1), ErrSchedulingDateTooEarly)
}

func TestClientFromPhoneNumbersRoundRobinLB(t *testing.T) {
	t.Parallel()

	stats := make(map[string]*uint64, len(client.(*sms).lb.PhoneNumbers())) //nolint:forcetypeassert // We know for sure.
	for _, number := range client.(*sms).lb.PhoneNumbers() {                //nolint:forcetypeassert // We know for sure.
		zero := uint64(0)
		stats[number] = &zero
	}

	const iterations = 10_000_000
	wg := new(sync.WaitGroup)
	wg.Add(iterations)
	for i := 0; i < iterations; i++ { //nolint:intrange // .
		go func() {
			defer wg.Done()
			atomic.AddUint64(stats[client.(*sms).lb.PhoneNumber()], 1) //nolint:forcetypeassert // We know for sure.
		}()
	}
	wg.Wait()
	for _, v := range stats {
		assert.InDelta(t, iterations/len(client.(*sms).lb.PhoneNumbers()), *v, 10) //nolint:forcetypeassert // We know for sure.
	}
}
