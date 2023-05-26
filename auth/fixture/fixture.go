// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"sync"
	stdlibtime "time"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

const (
	jwtIssuer   = "ice.io"
	defaultRole = "app"
)

//nolint:gochecknoglobals // We're using lazy stateless singletons for the whole testing runtime.
var (
	globalClient *firebaseAuth.Client
	singleton    = new(sync.Once)
)

type (
	CustomToken struct {
		*jwt.RegisteredClaims
		Custom   *map[string]any `json:"custom,omitempty"`
		Role     string          `json:"role" example:"1"`
		Email    string          `json:"email" example:"jdoe@example.com"`
		HashCode int64           `json:"hashCode,omitempty" example:"12356789"`
		Seq      int64           `json:"seq" example:"1"`
	}
)

func client() *firebaseAuth.Client {
	singleton.Do(func() {
		globalClient = internal.New(context.Background(), "_")
	})

	return globalClient
}

func CreateUser(role string) (uid, token string) {
	createCtx, cancelCreateUser := context.WithTimeout(context.Background(), 30*stdlibtime.Second) //nolint:gomnd // Not an issue here.
	defer cancelCreateUser()

	uid, email, password := generateUser(createCtx, role)

	req, err := json.MarshalContext(createCtx, map[string]any{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	})
	log.Panic(err) //nolint:revive // Intended.

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", os.Getenv("GCP_FIREBASE_AUTH_API_KEY"))
	respBytes := postRequest(url, req)

	var respBody struct {
		IDToken string `json:"idToken"`
	}
	log.Panic(json.UnmarshalContext(context.Background(), respBytes, &respBody))

	return uid, respBody.IDToken
}

func DeleteUser(uid string) {
	delCtx, cancelDeleteUser := context.WithTimeout(context.Background(), 30*stdlibtime.Second) //nolint:gomnd // Not an issue here.
	defer cancelDeleteUser()

	log.Panic(client().DeleteUser(delCtx, uid))
}

func GetUser(ctx context.Context, uid string) (*firebaseAuth.UserRecord, error) {
	return client().GetUser(ctx, uid) //nolint:wrapcheck // It's a proxy.
}

func generateUser(ctx context.Context, role string) (uid, email, password string) {
	const (
		phoneNumberMin = 1_000_000_000
		phoneNumberMax = 9_000_000_000
	)
	randNumber, err := rand.Int(rand.Reader, big.NewInt(phoneNumberMax-phoneNumberMin))
	log.Panic(err) //nolint:revive // Intended.
	phoneNumber := fmt.Sprintf("+1%d", randNumber.Uint64()+phoneNumberMin)
	password = uuid.NewString()

	user := new(firebaseAuth.UserToCreate).
		Email(fmt.Sprintf("%s@%s-test-user.com", uuid.NewString(), uuid.NewString())).
		Password(password).
		PhoneNumber(phoneNumber).
		UID(uuid.NewString()).
		DisplayName("test user")

	createdUser, err := client().CreateUser(ctx, user)
	log.Panic(err)

	log.Panic(client().SetCustomUserClaims(ctx, createdUser.UID, map[string]any{"role": role}))

	return createdUser.UID, createdUser.Email, password
}

func postRequest(url string, req []byte) []byte {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(req)) //nolint:gosec,noctx // it's a url for testing, so it can be variable
	log.Panic(err)                                                        //nolint:revive // Intended.
	defer func() {
		log.Panic(resp.Body.Close())
	}()
	if http.StatusOK != resp.StatusCode {
		log.Panic(errors.Errorf("unexpected status %v", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	log.Panic(err)

	return bodyBytes
}

func generateRefreshToken(now *time.Time, secret, userID, email string, seq int64, expiresAt stdlibtime.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresAt)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email: email,
		Seq:   seq,
	})
	refreshToken, err := token.SignedString([]byte(secret))

	return refreshToken, errors.Wrapf(err, "failed to generate refresh token for userID:%v, email:%v", userID, email)
}

func generateAccessToken(now *time.Time, refreshTokenSeq, hashCode int64, secret, userID, email string, expiresAt stdlibtime.Duration, claims map[string]any) (string, error) {
	var customClaims *map[string]any
	role := defaultRole
	customClaimsData := map[string]any(claims)
	if clRole, ok := customClaimsData["role"]; ok {
		role = clRole.(string)
		delete(customClaimsData, "role")
	}
	if len(customClaimsData) > 0 {
		customClaims = &customClaimsData
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresAt)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Role:     role,
		Email:    email,
		HashCode: hashCode,
		Seq:      refreshTokenSeq,
		Custom:   customClaims,
	})
	tokenStr, err := token.SignedString([]byte(secret))

	return tokenStr, errors.Wrapf(err, "failed to generate access token for userID:%v and email:%v", userID, email)
}

func GenerateTokens(now *time.Time, secret, userID, email string, hashCode, seq int64, expire stdlibtime.Duration, claims map[string]any) (refreshToken, accessToken string, err error) {
	refreshToken, err = generateRefreshToken(now, secret, userID, email, seq, expire)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to generate jwt refreshToken for userID:%v", userID)
	}
	accessToken, err = generateAccessToken(now, seq, hashCode, secret, userID, email, expire, claims)

	return refreshToken, accessToken, errors.Wrapf(err, "failed to generate accessToken for userID:%v", userID)
}
