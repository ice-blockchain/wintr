// SPDX-License-Identifier: ice License 1.0

package email

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/email/fixture"
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
	os.Exit(m.Run())
}

func TestClientSend(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	p1 := &Parcel{
		Body: &Body{
			Type: TextHTML,
			Data: "<strong>123456</strong>",
		},
		Subject: "Testing wintr/email",
		From: Participant{
			Name:  "ICE",
			Email: "no-reply@ice.io",
		},
	}
	testingEmail1 := fixture.TestingEmail(ctx, t)
	testingEmail2 := fixture.TestingEmail(ctx, t)
	require.NoError(t, client.Send(ctx, p1, Participant{
		Name:  "n1",
		Email: testingEmail1,
	},
		Participant{
			Name:   "n2",
			Email:  testingEmail2,
			SendAt: time.New(time.Now().Add(2 * stdlibtime.Second)),
		}))
	fixture.AssertEmailConfirmationCode(ctx, t, testingEmail1, "123456", func(msg string) string { return string([]byte(msg)[8:14]) })
	fixture.AssertEmailConfirmationCode(ctx, t, testingEmail2, "123456", func(msg string) string { return string([]byte(msg)[8:14]) })
}

func TestClientSendRetry(t *testing.T) { //nolint:paralleltest // We're testing ratelimit, we have 2 tests that need to not run in parallel.
	if true { // Remove this when testing locally.
		return
	}
	const rateLimit = 1
	wg := new(sync.WaitGroup)
	wg.Add(rateLimit)
	for i := 0; i < rateLimit; i++ { //nolint:intrange // .
		go requireSend(t, wg)
	}
	wg.Wait()
	wg.Add(rateLimit)
	for i := 0; i < rateLimit; i++ { //nolint:intrange // .
		go requireSend(t, wg)
	}
	wg.Wait()
}

func requireSend(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	defer wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	p1 := &Parcel{
		Body: &Body{
			Type: TextHTML,
			Data: "<strong>something</strong>",
		},
		Subject: "Testing wintr/email retry",
		From: Participant{
			Name:  "ICE",
			Email: "no-reply@ice.io",
		},
	}
	destinations := make([]Participant, 0, MaxBatchSize)
	for i := range MaxBatchSize {
		destinations = append(destinations, Participant{
			Name:  fmt.Sprintf("n%v", i),
			Email: fmt.Sprintf("foo%v@baz", i),
		})
	}
	assert.NoError(t, client.Send(ctx, p1, destinations...))
}
