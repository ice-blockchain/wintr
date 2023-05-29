// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"strings"
	"testing"
	stdlibtime "time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/auth/fixture"
	"github.com/ice-blockchain/wintr/auth/internal"
	"github.com/ice-blockchain/wintr/log"
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

type (
	mockIceConfiguration struct {
		JWTSecret string
		Expire    stdlibtime.Duration
	}
)

func (m mockIceConfiguration) SignedString(token *jwt.Token) (string, error) {
	return token.SignedString([]byte(m.JWTSecret)) //nolint:wrapcheck // .
}

func (m mockIceConfiguration) AccessDuration() stdlibtime.Duration {
	return m.Expire
}

func (m mockIceConfiguration) RefreshDuration() stdlibtime.Duration {
	return m.Expire
}

func (m mockIceConfiguration) Verify() func(token *jwt.Token) (any, error) {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if iss, err := token.Claims.GetIssuer(); err != nil || iss != JwtIssuer {
			return nil, errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
		}

		return []byte(m.JWTSecret), nil
	}
}

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

func TestVerifyFBToken_ValidToken(t *testing.T) {
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

func TestVerifyFBToken_InvalidToken(t *testing.T) {
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

func TestVerifyToken_Valid_FB_Token(t *testing.T) {
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

func TestVerifyIceToken_ValidToken(t *testing.T) { //nolint:paralleltest // .
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	mockedSecret := &mockIceConfiguration{JWTSecret: "1337", Expire: stdlibtime.Hour}
	var (
		au       = auth{ice: &authIce{secret: mockedSecret}}
		now      = time.Now()
		seq      = int64(0)
		hashCode = int64(0)
		userID   = "bogus"
		email    = "bogus@bogus.com"
		role     = "app"
		claims   = map[string]any{
			"role": role,
		}
	)
	refreshToken, accessToken, err := internal.GenerateTokens(mockedSecret, now, userID, email, hashCode, seq, claims)
	require.NoError(t, err)

	verifiedAccessToken, err := au.VerifyToken(ctx, accessToken)
	require.NoError(t, err)

	assert.Equal(t, email, verifiedAccessToken.Email)
	assert.Equal(t, role, verifiedAccessToken.Role)
	assert.Equal(t, userID, verifiedAccessToken.UserID)
	assert.Equal(t, email, verifiedAccessToken.Claims["email"])
	assert.Equal(t, seq, verifiedAccessToken.Claims["seq"])
	assert.Equal(t, role, verifiedAccessToken.Claims["role"])

	_, err = au.VerifyToken(ctx, refreshToken)
	require.Error(t, err, ErrWrongTypeToken)
}

func TestVerifyIceToken_WrongSecret(t *testing.T) { //nolint:paralleltest // .
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	mockedSecret := &mockIceConfiguration{JWTSecret: "1337", Expire: stdlibtime.Hour}
	var (
		au       = auth{ice: &authIce{secret: mockedSecret}}
		seq      = int64(0)
		hashCode = int64(0)
		userID   = "bogus"
		email    = "bogus@bogus.com"
		claims   = map[string]any{
			"role": "author",
		}
		now = time.Now()
	)
	refreshToken, accessToken, err := internal.GenerateTokens(mockedSecret, now, userID, email, hashCode, seq, claims)
	require.NoError(t, err)

	mockedSecret.JWTSecret = "another_secret"
	token, err := au.VerifyToken(ctx, accessToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, jwt.ErrSignatureInvalid)
	assert.Nil(t, token)

	token, err = au.VerifyToken(ctx, refreshToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, jwt.ErrSignatureInvalid)
	assert.Nil(t, token)
}

func TestVerifyIceToken_TokenExpired(t *testing.T) { //nolint:paralleltest // config is loaded only once.
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	mockedSecret := &mockIceConfiguration{JWTSecret: "1337", Expire: stdlibtime.Duration(0)}
	var (
		au       = auth{ice: &authIce{secret: mockedSecret}}
		now      = time.Now()
		seq      = int64(0)
		hashCode = int64(0)
		userID   = "bogus"
		email    = "bogus@bogus.com"
		claims   = map[string]any{"role": "author"}
	)
	refreshToken, accessToken, err := internal.GenerateTokens(mockedSecret, now, userID, email, hashCode, seq, claims)
	require.NoError(t, err)
	token, err := au.VerifyToken(ctx, accessToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, token)
	token, err = au.VerifyToken(ctx, refreshToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, token)
}

func TestVerifyIceToken_WrongSigningMethod(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	m := &mockIceConfiguration{JWTSecret: "123456789", Expire: stdlibtime.Hour}
	var (
		au        = auth{ice: &authIce{secret: m}, fb: client}
		now       = time.Now().In(stdlibtime.UTC)
		jwtSecret = "123456789" //nolint:gosec // .
		userID    = "bogus"
		expire    = stdlibtime.Hour
	)
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, IceToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    JwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	iceToken, err := au.VerifyToken(ctx, tokenStr)
	require.Error(t, err)
	assert.Nil(t, iceToken)
}

func TestVerifyIceToken_Parse(t *testing.T) {
	t.Parallel()
	mockedSecret := &mockIceConfiguration{JWTSecret: "1337", Expire: stdlibtime.Duration(0)}
	var (
		now      = time.Now()
		seq      = int64(0)
		hashCode = int64(0)
		userID   = "bogus"
		email    = "bogus@bogus.com"
		claims   = map[string]any{
			"role": "author",
		}
	)
	refreshToken, accessToken, err := internal.GenerateTokens(mockedSecret, now, userID, email, hashCode, seq, claims)
	require.NoError(t, err)
	err = detectIceToken(accessToken)
	require.NoError(t, err)
	err = detectIceToken(refreshToken)
	require.NoError(t, err)
}

func TestDetectIceToken_WrongToken(t *testing.T) {
	t.Parallel()
	token := "dummy token" //nolint:gosec // .
	err := detectIceToken(token)
	require.Error(t, err)
}

func TestDetectIceToken_WrongIssuer(t *testing.T) { //nolint:funlen // config is loaded only once.
	t.Parallel()
	var (
		now       = time.Now()
		seq       = int64(0)
		hashCode  = int64(0)
		userID    = "bogus"
		role      = "author"
		email     = "bogus@bogus.com"
		jwtSecret = "1337" //nolint:gosec // .
		expire    = stdlibtime.Hour
		claims    = map[string]any{
			"role": "author",
		}
	)
	authToken := jwt.NewWithClaims(jwt.SigningMethodHS256, IceToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    "wrong issue",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email:    email,
		HashCode: hashCode,
		Role:     role,
		Seq:      seq,
		Custom:   &claims,
	})
	tokenStr, err := authToken.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	err = detectIceToken(tokenStr)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestDetectIceToken_WrongAlgorithmMethod(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	var (
		now      = time.Now()
		bitSize  = 4096
		seq      = int64(0)
		hashCode = int64(0)
		userID   = "bogus"
		role     = "author"
		email    = "bogus@bogus.com"
		expire   = stdlibtime.Hour
		claims   = map[string]any{
			"role": "author",
		}
	)
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	require.NoError(t, err)

	authToken := jwt.NewWithClaims(jwt.SigningMethodRS256, IceToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    JwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email:    email,
		HashCode: hashCode,
		Role:     role,
		Seq:      seq,
		Custom:   &claims,
	})
	tokenStr, err := authToken.SignedString(key)
	require.NoError(t, err)
	err = detectIceToken(tokenStr)
	require.Error(t, err)
}

func TestDetectIceToken_WrongAlgorithmLength(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	var (
		seq       = int64(0)
		hashCode  = int64(0)
		userID    = "bogus"
		role      = "author"
		email     = "bogus@bogus.com"
		jwtSecret = "1337" //nolint:gosec // .
		expire    = stdlibtime.Hour
		claims    = map[string]any{
			"role": "author",
		}
	)
	now := time.Now().In(stdlibtime.UTC)
	authToken := jwt.NewWithClaims(jwt.SigningMethodHS384, IceToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    JwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Email:    email,
		HashCode: hashCode,
		Role:     role,
		Seq:      seq,
		Custom:   &claims,
	})
	tokenStr, err := authToken.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	err = detectIceToken(tokenStr)
	require.Error(t, err)
}
