// SPDX-License-Identifier: ice License 1.0

package firebaseauth

import (
	"context"
	"fmt"
	"os"
	"strings"
	stdlibtime "time"

	"dario.cat/mergo"
	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/pkg/errors"
	firebaseoption "google.golang.org/api/option"

	"github.com/ice-blockchain/wintr/auth/internal"
	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

//nolint:funlen // .
func New(ctx context.Context, applicationYAMLKey string) Client {
	cfg := new(config)
	appcfg.MustLoadFromKey(applicationYAMLKey, cfg)
	cfg.setWintrAuthFirebaseCredentialsFileContent(applicationYAMLKey)
	cfg.setWintrAuthFirebaseCredentialsFilePath(applicationYAMLKey)

	var credentialsOption firebaseoption.ClientOption
	if cfg.WintrAuthFirebase.Credentials.FileContent != "" {
		credentialsOption = firebaseoption.WithCredentialsJSON([]byte(cfg.WintrAuthFirebase.Credentials.FileContent))
	}
	if cfg.WintrAuthFirebase.Credentials.FilePath != "" {
		credentialsOption = firebaseoption.WithCredentialsFile(cfg.WintrAuthFirebase.Credentials.FilePath)
	}
	if credentialsOption == nil {
		log.Info("firebase auth client is not initialized, because it is not configured")

		return nil
	}
	firebaseApp, err := firebase.NewApp(ctx, nil, credentialsOption)
	log.Panic(errors.Wrapf(err, "[%v] failed to build Firebase app ", applicationYAMLKey)) //nolint:revive // That's intended.
	client, err := firebaseApp.Auth(ctx)
	log.Panic(errors.Wrapf(err, "[%v] failed to build Firebase Auth client", applicationYAMLKey))

	eagerLoadCtx, cancelEagerLoad := context.WithTimeout(ctx, 30*stdlibtime.Second) //nolint:mnd,gomnd // It's a one time call.
	defer cancelEagerLoad()
	t, err := client.VerifyIDTokenAndCheckRevoked(eagerLoadCtx, "invalid token")
	if t != nil || !firebaseauth.IsIDTokenInvalid(err) {
		log.Panic(errors.New("unexpected success"))
	}

	return &auth{
		client:             client,
		allowEmailPassword: cfg.WintrAuthFirebase.AllowEmailPassword,
	}
}

func (a *auth) VerifyToken(ctx context.Context, token string) (*internal.Token, error) {
	firebaseToken, vErr := a.client.VerifyIDToken(ctx, token)
	if vErr != nil {
		return nil, errors.Wrap(vErr, "error verifying firebase token")
	}
	if (!a.allowEmailPassword) && firebaseToken.Firebase.SignInProvider == passwordSignInProvider {
		emailVerified := false
		if emailVerifiedInterface, found := firebaseToken.Claims["email_verified"]; found {
			emailVerified, _ = emailVerifiedInterface.(bool) //nolint:errcheck,revive // Not needed.
		}
		if !emailVerified {
			return nil, errors.Wrapf(ErrForbidden, "%v sign_in_provider is not allowed without verified email", firebaseToken.Firebase.SignInProvider)
		}
	}
	var email, role string
	userID := firebaseToken.UID
	if len(firebaseToken.Claims) > 0 {
		if emailInterface, found := firebaseToken.Claims["email"]; found {
			email, _ = emailInterface.(string) //nolint:errcheck,revive // Not needed.
		}
		if roleInterface, found := firebaseToken.Claims["role"]; found {
			role, _ = roleInterface.(string) //nolint:errcheck,revive // Not needed.
		}
	}

	return &internal.Token{
		UserID:   userID,
		Claims:   firebaseToken.Claims,
		Email:    email,
		Role:     role,
		Provider: internal.ProviderFirebase,
	}, nil
}

func (a *auth) UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user, err := a.client.GetUser(ctx, userID)
	if err != nil {
		if strings.HasSuffix(err.Error(), fmt.Sprintf("no user exists with the uid: %q", userID)) {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to get current user for userID:`%v`", userID)
	}
	if err = mergo.Merge(&customClaims, user.CustomClaims, mergo.WithOverride, mergo.WithTypeCheck); err != nil {
		return errors.Wrapf(err, "failed to merge %#v and %#v", customClaims, user.CustomClaims)
	}
	if err = a.client.SetCustomUserClaims(ctx, userID, customClaims); err != nil {
		if strings.HasSuffix(err.Error(), "no user record found for the given identifier") {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to update custom claims to `%#v`, for userID:`%v`", customClaims, userID)
	}

	return nil
}

func (a *auth) UpdateEmail(ctx context.Context, userID, email string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if _, err := a.client.UpdateUser(ctx, userID, new(firebaseauth.UserToUpdate).Email(email).EmailVerified(true)); err != nil {
		if strings.HasSuffix(err.Error(), "user with the provided email already exists") {
			return ErrConflict
		}
		if strings.HasSuffix(err.Error(), "no user record found for the given identifier") {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to update email to `%v`, for userID:`%v`", email, userID)
	}

	return nil
}

func (a *auth) DeleteUser(ctx context.Context, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if err := a.client.DeleteUser(ctx, userID); err != nil {
		if err.Error() == "no user record found for the given identifier" {
			return nil
		}

		return errors.Wrapf(err, "failed to delete user by ID:`%v`", userID)
	}

	return nil
}

func (a *auth) GetUser(ctx context.Context, userID string) (*firebaseauth.UserRecord, error) {
	usr, err := a.client.GetUser(ctx, userID)

	return usr, errors.Wrapf(err, "can't get firebase user data for:%v", userID)
}

func (a *auth) GetUserByEmail(ctx context.Context, email string) (*firebaseauth.UserRecord, error) {
	usr, err := a.client.GetUserByEmail(ctx, email)
	if err != nil {
		if strings.HasSuffix(err.Error(), fmt.Sprintf("no user exists with the email: \"%v\"", email)) {
			return nil, ErrUserNotFound
		}

		return nil, errors.Wrapf(err, "can't get firebase user data by email for:%v", email)
	}

	return usr, nil
}

func (cfg *config) setWintrAuthFirebaseCredentialsFileContent(applicationYAMLKey string) {
	if cfg.WintrAuthFirebase.Credentials.FileContent == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrAuthFirebase.Credentials.FileContent = os.Getenv(module + "_AUTH_CREDENTIALS_FILE_CONTENT")
		if cfg.WintrAuthFirebase.Credentials.FileContent == "" {
			cfg.WintrAuthFirebase.Credentials.FileContent = os.Getenv("AUTH_CREDENTIALS_FILE_CONTENT")
		}
		if cfg.WintrAuthFirebase.Credentials.FileContent == "" {
			cfg.WintrAuthFirebase.Credentials.FileContent = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if !strings.HasPrefix(strings.TrimSpace(cfg.WintrAuthFirebase.Credentials.FileContent), "{") {
				cfg.WintrAuthFirebase.Credentials.FileContent = ""
			}
		}
	}
}

func (cfg *config) setWintrAuthFirebaseCredentialsFilePath(applicationYAMLKey string) {
	if cfg.WintrAuthFirebase.Credentials.FilePath == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrAuthFirebase.Credentials.FilePath = os.Getenv(module + "_AUTH_CREDENTIALS_FILE_PATH")
		if cfg.WintrAuthFirebase.Credentials.FilePath == "" {
			cfg.WintrAuthFirebase.Credentials.FilePath = os.Getenv("AUTH_CREDENTIALS_FILE_PATH")
		}
		if cfg.WintrAuthFirebase.Credentials.FilePath == "" {
			cfg.WintrAuthFirebase.Credentials.FilePath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if strings.HasPrefix(strings.TrimSpace(cfg.WintrAuthFirebase.Credentials.FilePath), "{") {
				cfg.WintrAuthFirebase.Credentials.FilePath = ""
			}
		}
	}
}
