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
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	"github.com/ice-blockchain/wintr/log"
)

//nolint:gochecknoglobals // We're using lazy stateless singletons for the whole testing runtime.
var (
	globalClient *firebaseAuth.Client
	singleton    = new(sync.Once)
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
