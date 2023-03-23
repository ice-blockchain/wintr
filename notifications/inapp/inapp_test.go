// SPDX-License-Identifier: ice License 1.0

package inapp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/notifications/inapp/fixture"
	"github.com/ice-blockchain/wintr/notifications/inapp/internal"
)

const (
	testApplicationYAMLKey   = "self"
	testNotificationFeedName = "testing"
	testFlatFeedName         = "testing2"
)

// .
var (
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	notificationFeedClient Client
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	flatFeedClient Client
)

func TestMain(m *testing.M) {
	notificationFeedClient = New(testApplicationYAMLKey, testNotificationFeedName)
	flatFeedClient = New(testApplicationYAMLKey, testFlatFeedName)
	os.Exit(m.Run())
}

func TestClientSend(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	userID := uuid.NewString()
	p1 := &Parcel{
		Data:   map[string]any{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Action: "join_team",
		Actor: internal.ID{
			Type:  "userId",
			Value: uuid.NewString(),
		},
		Subject: internal.ID{
			Type:  "userId",
			Value: userID,
		},
	}
	require.NoError(t, notificationFeedClient.Send(ctx, p1, userID))
	require.NoError(t, flatFeedClient.Send(ctx, p1, userID))
	p2 := &Parcel{
		Data:   map[string]any{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Action: "join_team",
		Actor: internal.ID{
			Type:  "userId",
			Value: uuid.NewString(),
		},
		Subject: internal.ID{
			Type:  "userId",
			Value: userID,
		},
	}
	require.NoError(t, notificationFeedClient.Send(ctx, p2, userID))
	require.NoError(t, flatFeedClient.Send(ctx, p2, userID))

	actual, err := fixture.GetAllInAppNotifications(ctx, testApplicationYAMLKey, testNotificationFeedName, userID)
	require.NoError(t, err)
	require.Len(t, actual, 2)
	assertInDelta(t, p1.Time.UnixNano(), actual[1].Time.UnixNano(), 1000)
	assertInDelta(t, p2.Time.UnixNano(), actual[0].Time.UnixNano(), 1000)
	actual[1].Time = p1.Time
	actual[0].Time = p2.Time
	assert.EqualValues(t, p1, actual[1]) // Because they're returned in the reverse order, from the latest to the oldest.
	assert.EqualValues(t, p2, actual[0])
	actual, err = fixture.GetAllInAppNotifications(ctx, testApplicationYAMLKey, testFlatFeedName, userID)
	require.NoError(t, err)
	require.Len(t, actual, 2)
	assertInDelta(t, p1.Time.UnixNano(), actual[1].Time.UnixNano(), 1000)
	assertInDelta(t, p2.Time.UnixNano(), actual[0].Time.UnixNano(), 1000)
	actual[1].Time = p1.Time
	actual[0].Time = p2.Time
	assert.EqualValues(t, p1, actual[1]) // Because they're returned in the reverse order, from the latest to the oldest.
	assert.EqualValues(t, p2, actual[0])
}

func TestClientBroadcast(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	p1 := &Parcel{
		Data:   map[string]any{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Action: "broadcast_news",
		Actor: internal.ID{
			Type:  "system",
			Value: "ice.io",
		},
		Subject: internal.ID{
			Type:  "newsId",
			Value: uuid.NewString(),
		},
	}
	u1, u2 := uuid.NewString(), uuid.NewString()
	require.NoError(t, notificationFeedClient.Send(ctx, p1, u1, u2))
	require.NoError(t, flatFeedClient.Send(ctx, p1, u1, u2))

	actual, err := fixture.GetAllInAppNotifications(ctx, testApplicationYAMLKey, testNotificationFeedName, u1)
	require.NoError(t, err)
	require.Len(t, actual, 1)
	assertInDelta(t, p1.Time.UnixNano(), actual[0].Time.UnixNano(), 1000)
	actual[0].Time = p1.Time
	assert.EqualValues(t, p1, actual[0])

	actual, err = fixture.GetAllInAppNotifications(ctx, testApplicationYAMLKey, testNotificationFeedName, u2)
	require.NoError(t, err)
	require.Len(t, actual, 1)
	assertInDelta(t, p1.Time.UnixNano(), actual[0].Time.UnixNano(), 1000)
	actual[0].Time = p1.Time
	assert.EqualValues(t, p1, actual[0])

	actual, err = fixture.GetAllInAppNotifications(ctx, testApplicationYAMLKey, testFlatFeedName, u1)
	require.NoError(t, err)
	require.Len(t, actual, 1)
	assertInDelta(t, p1.Time.UnixNano(), actual[0].Time.UnixNano(), 1000)
	actual[0].Time = p1.Time
	assert.EqualValues(t, p1, actual[0])

	actual, err = fixture.GetAllInAppNotifications(ctx, testApplicationYAMLKey, testFlatFeedName, u2)
	require.NoError(t, err)
	require.Len(t, actual, 1)
	assertInDelta(t, p1.Time.UnixNano(), actual[0].Time.UnixNano(), 1000)
	actual[0].Time = p1.Time
	assert.EqualValues(t, p1, actual[0])
}

func TestClientCreateUserToken(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	token, err := notificationFeedClient.CreateUserToken(ctx, uuid.NewString())
	require.NoError(t, err)
	require.NotEmpty(t, token.AppID)
	require.NotEmpty(t, token.APIKey)
	require.NotEmpty(t, token.APISecret)
}

func TestClientBroadcastRetry(t *testing.T) { //nolint:paralleltest // We're testing ratelimit, we have 2 tests that need to not run in parallel.
	if true { // Remove this when testing locally.
		return
	}
	const rateLimit = 120
	wg := new(sync.WaitGroup)
	wg.Add(rateLimit)
	for i := 0; i < rateLimit; i++ {
		go requireBroadcast(t, wg)
	}
	wg.Wait()
	stdlibtime.Sleep(40 * stdlibtime.Second)
	wg.Add(rateLimit)
	for i := 0; i < rateLimit; i++ {
		go requireBroadcast(t, wg)
	}
	wg.Wait()
	stdlibtime.Sleep(1 * stdlibtime.Minute)
}

func requireBroadcast(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	defer wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	p1 := &Parcel{
		Data:   map[string]any{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Action: "broadcast_news",
		Actor: internal.ID{
			Type:  "system",
			Value: "ice.io",
		},
		Subject: internal.ID{
			Type:  "newsId",
			Value: uuid.NewString(),
		},
	}
	userIDs := make([]UserID, MaxBatchSize, MaxBatchSize) //nolint:gosimple // Prefer to be more descriptive.
	for i := range userIDs {
		userIDs[i] = uuid.NewString()
	}
	require.NoError(t, notificationFeedClient.Send(ctx, p1, userIDs...))
}

func TestClientSendRetry(t *testing.T) { //nolint:paralleltest // We're testing ratelimit, we have 2 tests that need to not run in parallel.
	if true { // Remove this when testing locally.
		return
	}
	const rateLimit = 1000
	wg := new(sync.WaitGroup)
	wg.Add(rateLimit)
	for i := 0; i < rateLimit; i++ {
		go requireSend(t, wg)
	}
	wg.Wait()
	stdlibtime.Sleep(40 * stdlibtime.Second)
	wg.Add(rateLimit)
	for i := 0; i < rateLimit; i++ {
		go requireSend(t, wg)
	}
	wg.Wait()
	stdlibtime.Sleep(1 * stdlibtime.Minute)
}

func requireSend(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	defer wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	userID := uuid.NewString()
	p1 := &Parcel{
		Data:   map[string]any{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Action: "join_team",
		Actor: internal.ID{
			Type:  "userId",
			Value: uuid.NewString(),
		},
		Subject: internal.ID{
			Type:  "userId",
			Value: userID,
		},
	}
	require.NoError(t, notificationFeedClient.Send(ctx, p1, userID))
}

func assertInDelta(tb testing.TB, expected, actual, delta int64) { //nolint:unparam // Not a problem.
	tb.Helper()
	diff := expected - actual
	assert.True(tb, diff <= delta, "diff is %v", diff)
	assert.True(tb, diff >= 0, "diff is %v", diff)
}
