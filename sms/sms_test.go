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
	"github.com/ice-blockchain/wintr/sms/internal"
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
	var messageService *internal.PhoneNumbersRoundRobinLB
	//nolint:forcetypeassert // .
	for _, sender := range client.(*sms).sendersByCountry {
		messageService = sender

		break
	}
	stats := make(map[string]*uint64, len(messageService.PhoneNumbers()))
	for _, number := range messageService.PhoneNumbers() {
		zero := uint64(0)
		stats[number] = &zero
	}

	const iterations = 10_000_000
	wg := new(sync.WaitGroup)
	wg.Add(iterations)
	for i := 0; i < iterations; i++ { //nolint:intrange // .
		go func() {
			defer wg.Done()
			atomic.AddUint64(stats[messageService.PhoneNumber()], 1)
		}()
	}
	wg.Wait()
	for _, v := range stats {
		assert.InDelta(t, iterations/len(messageService.PhoneNumbers()), *v, 10)
	}
}

//nolint:funlen // Test cases.
func TestDetectCounty(t *testing.T) {
	t.Parallel()
	tests := []*struct {
		name            string
		phoneNumber     string
		expectedCountry string
		expectError     bool
	}{
		{
			name:            "US phone number full format",
			phoneNumber:     "+12125551234",
			expectedCountry: "us",
			expectError:     false,
		},
		{
			name:            "UK phone number with +44 prefix",
			phoneNumber:     "+447777123456",
			expectedCountry: "gb",
			expectError:     false,
		},
		{
			name:            "Germany phone number with +49 prefix",
			phoneNumber:     "+4930123456789",
			expectedCountry: "de",
			expectError:     false,
		},
		{
			name:            "France phone number with +33 prefix",
			phoneNumber:     "+33123456789",
			expectedCountry: "fr",
			expectError:     false,
		},
		{
			name:            "Japan phone number with +81 prefix",
			phoneNumber:     "+81312345678",
			expectedCountry: "jp",
			expectError:     false,
		},
		{
			name:            "China phone number with +86 prefix",
			phoneNumber:     "+8613812345678",
			expectedCountry: "cn",
			expectError:     false,
		},
		{
			name:            "Brazil phone number with +55 prefix",
			phoneNumber:     "+5511987654321",
			expectedCountry: "br",
			expectError:     false,
		},
		{
			name:            "India phone number with +91 prefix",
			phoneNumber:     "+919876543210",
			expectedCountry: "in",
			expectError:     false,
		},
		{
			name:        "Invalid phone number - empty string",
			phoneNumber: "",
			expectError: true,
		},
		{
			name:        "Invalid phone number - no country code",
			phoneNumber: "1234567890",
			expectError: true,
		},
		{
			name:        "Invalid phone number - malformed",
			phoneNumber: "invalid-phone",
			expectError: true,
		},
		{
			name:        "Invalid phone number - too short",
			phoneNumber: "+1",
			expectError: true,
		},
		{
			name:        "Invalid phone number - special characters",
			phoneNumber: "+1@#$%^&*()",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			country, err := detectCounty(tt.phoneNumber)

			if tt.expectError {
				require.Error(t, err, "Expected error for phone number: %s", tt.phoneNumber)

				return
			}
			require.NoError(t, err, "Unexpected error for phone number: %s", tt.phoneNumber)
			assert.Equal(t, tt.expectedCountry, country,
				"Country mismatch for phone number: %s", tt.phoneNumber)
			assert.NotEmpty(t, country, "Country should not be empty for valid phone number")
		})
	}
}
