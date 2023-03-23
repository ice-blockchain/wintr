// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"
	"os"
	"strings"
	"testing"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/auth/fixture"
	"github.com/ice-blockchain/wintr/log"
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*stdlibtime.Second)
	client = New(ctx, testApplicationYAMLKey)
	defer func() {
		if e := recover(); e != nil {
			cancel()
			log.Panic(e)
		}
	}()
	exitCode := m.Run()
	cancel()
	os.Exit(exitCode) //nolint:gocritic // That's intended.
}

func TestVerifyToken_ValidToken(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	uid, idToken := fixture.CreateUser("app")
	defer fixture.DeleteUser(uid)

	token, err := client.VerifyToken(ctx, idToken)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotEmpty(t, token.UserID)
	require.Equal(t, uid, token.UserID)
	require.NotEmpty(t, token.Email)
	require.Equal(t, "app", token.Role)
	require.NotEmpty(t, token.Claims)
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	token, err := client.VerifyToken(ctx, "invalid token")
	require.Error(t, err)
	require.Nil(t, token)
}

func TestUpdateEmail_Success(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	uid, _ := fixture.CreateUser("app")
	uid2, _ := fixture.CreateUser("app")
	defer fixture.DeleteUser(uid)
	defer fixture.DeleteUser(uid2)

	user, err := fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	user2, err := fixture.GetUser(ctx, uid2)
	require.NoError(t, err)
	require.NotEqual(t, "foo1@bar.com", user.Email)
	require.False(t, user.EmailVerified)
	require.NoError(t, client.UpdateEmail(ctx, uid, "foo1@bar.com"))
	require.ErrorIs(t, client.UpdateEmail(ctx, uid, user2.Email), ErrConflict)
	require.ErrorIs(t, client.UpdateEmail(ctx, uuid.NewString(), "foo1@bar.com"), ErrUserNotFound)
	user, err = fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	require.Equal(t, "foo1@bar.com", user.Email)
	require.True(t, user.EmailVerified)
}

func TestUpdatePhoneNumber_Success(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	uid, _ := fixture.CreateUser("app")
	uid2, _ := fixture.CreateUser("app")
	defer fixture.DeleteUser(uid)
	defer fixture.DeleteUser(uid2)

	user, err := fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	user2, err := fixture.GetUser(ctx, uid2)
	require.NoError(t, err)
	require.NotEqual(t, "+12345678900", user.PhoneNumber)
	require.NoError(t, client.UpdatePhoneNumber(ctx, uid, "+12345678900"))
	require.ErrorIs(t, client.UpdatePhoneNumber(ctx, uid, user2.PhoneNumber), ErrConflict)
	require.ErrorIs(t, client.UpdatePhoneNumber(ctx, uuid.NewString(), "+12345678901"), ErrUserNotFound)
	user, err = fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	require.Equal(t, "+12345678900", user.PhoneNumber)
}

func TestUpdateCustomClaims_Success(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	uid, _ := fixture.CreateUser("app")
	defer fixture.DeleteUser(uid)

	user, err := fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	require.EqualValues(t, map[string]any{"role": "app"}, user.CustomClaims)
	require.NoError(t, client.UpdateCustomClaims(ctx, uid, map[string]any{"a": 1, "b": map[string]any{"c": "x"}}))
	require.NoError(t, client.UpdateCustomClaims(ctx, uid, map[string]any{"b": map[string]any{"d": "y"}}))
	require.ErrorIs(t, client.UpdateCustomClaims(ctx, uuid.NewString(), map[string]any{"a": 1}), ErrUserNotFound)
	user, err = fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	require.EqualValues(t, map[string]any{"a": 1.0, "b": map[string]any{"c": "x", "d": "y"}, "role": "app"}, user.CustomClaims)
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	uid, _ := fixture.CreateUser("app")

	user, err := fixture.GetUser(ctx, uid)
	require.NoError(t, err)
	require.NotEmpty(t, user.PhoneNumber)
	require.NoError(t, client.DeleteUser(ctx, uid))
	require.NoError(t, client.DeleteUser(ctx, uuid.NewString()), ErrUserNotFound)
	_, err = fixture.GetUser(ctx, uid)
	require.NotNil(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "no user exists with the"))
}
