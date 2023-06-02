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

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	firebaseoption "google.golang.org/api/option"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

//nolint:gochecknoglobals // We're using lazy stateless singletons for the whole testing runtime.
var (
	globalFirebaseClient *firebaseAuth.Client
	singleton            = new(sync.Once)
)

func client() *firebaseAuth.Client {
	singleton.Do(func() {
		globalFirebaseClient = newFirebaseClient()
	})

	return globalFirebaseClient
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

func SetCustomUserClaims(ctx context.Context, uid string, claims map[string]any) error {
	err := client().SetCustomUserClaims(ctx, uid, claims)
	log.Panic(err, "can't set custom user claims")

	return err //nolint:wrapcheck // .
}

func GenerateIceTokens(userID, role string) (refreshToken, accessToken string, err error) {
	var (
		client = fixtureIceAuth{
			RefreshExpirationTime: 12 * stdlibtime.Hour, //nolint:gomnd // It's just hours.
			AccessExpirationTime:  12 * stdlibtime.Hour, //nolint:gomnd // It's just hours.
		}
		now      = time.Now()
		email    = uuid.NewString() + "@testuser.com"
		seq      = int64(0)
		hashCode = int64(0)
		claims   = map[string]any{"role": role}
	)
	refreshToken, err = client.generateIceRefreshToken(now, userID, email, seq)
	if err != nil {
		return "", "", err
	}
	accessToken, err = client.generateIceAccessToken(now, seq, hashCode, userID, email, claims)

	return refreshToken, accessToken, err
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

	usr, err := client().CreateUser(ctx, user)
	log.Panic(err, "can't create user")
	log.Panic(SetCustomUserClaims(ctx, usr.UID, map[string]any{"role": role}))

	return usr.UID, usr.Email, password
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

func (f *fixtureIceAuth) generateIceRefreshToken(now *time.Time, userID, email string, seq int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, IceToken{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(f.RefreshExpirationTime)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email: email,
		Seq:   seq,
	})

	return signedString(token)
}

//nolint:revive // Fields.
func (f *fixtureIceAuth) generateIceAccessToken(
	now *time.Time, refreshTokenSeq, hashCode int64,
	userID, email string,
	claims map[string]any,
) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, IceToken{ //nolint:forcetypeassert // .
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(f.AccessExpirationTime)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Role:     claims["role"].(string),
		Email:    email,
		HashCode: hashCode,
		Seq:      refreshTokenSeq,
		Custom:   &claims,
	})

	return signedString(token)
}

func signedString(token *jwt.Token) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" { //nolint:gosec // .
		log.Panic("jwt secret not provided")
	}

	return token.SignedString([]byte(jwtSecret)) //nolint:wrapcheck // .
}

func newFirebaseClient() *firebaseAuth.Client {
	ctx := context.Background()
	fileContent := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	filePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	var credentialsOption firebaseoption.ClientOption
	if fileContent != "" { //nolint:gocritic // Wrong.
		credentialsOption = firebaseoption.WithCredentialsJSON([]byte(fileContent))
	} else if filePath != "" {
		credentialsOption = firebaseoption.WithCredentialsFile(filePath)
	} else {
		log.Panic("can't find credentials")
	}
	firebaseApp, err := firebase.NewApp(ctx, nil, credentialsOption)
	log.Panic(errors.Wrap(err, "[%v] failed to build Firebase app ")) //nolint:revive // That's intended.
	client, err := firebaseApp.Auth(ctx)
	log.Panic(errors.Wrap(err, "[%v] failed to build Firebase Auth client"))

	eagerLoadCtx, cancelEagerLoad := context.WithTimeout(ctx, 5*stdlibtime.Second) //nolint:gomnd // It's a one time call.
	defer cancelEagerLoad()
	t, err := client.VerifyIDTokenAndCheckRevoked(eagerLoadCtx, "invalid token")
	if t != nil || !firebaseAuth.IsIDTokenInvalid(err) {
		log.Panic(errors.New("unexpected success"))
	}

	return client
}
