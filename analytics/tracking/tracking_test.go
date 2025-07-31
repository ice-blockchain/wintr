// SPDX-License-Identifier: ice License 1.0

package tracking

import (
	"context"
	"fmt"
	"os"
	"testing"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	testApplicationYAMLKey = "self"
)

// .
var (
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	client *tracking
)

func TestMain(m *testing.M) {
	client = New(testApplicationYAMLKey).(*tracking) //nolint:forcetypeassert,revive,errcheck // We know for sure.
	os.Exit(m.Run())                                 //nolint:revive // .
}

func TestClientTrackAction(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	require.NoError(t, client.TrackAction(ctx, "bogus", &Action{
		Name: "test",
		Attributes: map[string]any{
			"bogus": 1,
		},
	}))
}

func TestClientSetUserAttributes(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	require.NoError(t, client.SetUserAttributes(ctx, "bogus", map[string]any{
		"bogus2": "test",
	}))
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	userID := client.createTestUser(ctx, t)
	require.NoError(t, client.DeleteUser(ctx, userID))
}

func (t *tracking) createTestUser(ctx context.Context, tt *testing.T) (userID string) { //nolint:thelper // It's a clash with the receiver.
	tt.Helper()
	userID = uuid.NewString()
	body := fmt.Sprintf(`{"type":"customer","customer_id":%q,"attributes":{"name":"John","platforms":[{"platform":"ANDROID","active":"true"}]}}`, userID)
	require.NoError(tt, t.post(ctx, t.cfg.Tracking.BaseURL+"/v1/customer/"+t.cfg.Tracking.Credentials.AppID, body))

	return userID
}
