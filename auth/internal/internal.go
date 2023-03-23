// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
	stdlibtime "time"

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/pkg/errors"
	firebaseoption "google.golang.org/api/option"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(ctx context.Context, applicationYAMLKey string) *firebaseAuth.Client {
	cfg := new(config)
	appCfg.MustLoadFromKey(applicationYAMLKey, cfg)
	cfg.setWintrServerAuthCredentialsFileContent(applicationYAMLKey)
	cfg.setWintrServerAuthCredentialsFilePath(applicationYAMLKey)

	var credentialsOption firebaseoption.ClientOption
	if cfg.WintrServerAuth.Credentials.FileContent != "" {
		credentialsOption = firebaseoption.WithCredentialsJSON([]byte(cfg.WintrServerAuth.Credentials.FileContent))
	}
	if cfg.WintrServerAuth.Credentials.FilePath != "" {
		credentialsOption = firebaseoption.WithCredentialsFile(cfg.WintrServerAuth.Credentials.FilePath)
	}
	firebaseApp, err := firebase.NewApp(ctx, nil, credentialsOption)
	log.Panic(errors.Wrapf(err, "[%v] failed to build Firebase app ", applicationYAMLKey)) //nolint:revive // That's intended.
	client, err := firebaseApp.Auth(ctx)
	log.Panic(errors.Wrapf(err, "[%v] failed to build Firebase Auth client", applicationYAMLKey))

	eagerLoadCtx, cancelEagerLoad := context.WithTimeout(ctx, 30*stdlibtime.Second) //nolint:gomnd // It's a one time call.
	defer cancelEagerLoad()
	t, err := client.VerifyIDTokenAndCheckRevoked(eagerLoadCtx, "invalid token")
	if t != nil || !firebaseAuth.IsIDTokenInvalid(err) {
		log.Panic(errors.New("unexpected success"))
	}

	return client
}

func (cfg *config) setWintrServerAuthCredentialsFileContent(applicationYAMLKey string) {
	if cfg.WintrServerAuth.Credentials.FileContent == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrServerAuth.Credentials.FileContent = os.Getenv(fmt.Sprintf("%s_AUTH_CREDENTIALS_FILE_CONTENT", module))
		if cfg.WintrServerAuth.Credentials.FileContent == "" {
			cfg.WintrServerAuth.Credentials.FileContent = os.Getenv("AUTH_CREDENTIALS_FILE_CONTENT")
		}
		if cfg.WintrServerAuth.Credentials.FileContent == "" {
			cfg.WintrServerAuth.Credentials.FileContent = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if !strings.HasPrefix(strings.TrimSpace(cfg.WintrServerAuth.Credentials.FileContent), "{") {
				cfg.WintrServerAuth.Credentials.FileContent = ""
			}
		}
	}
}

func (cfg *config) setWintrServerAuthCredentialsFilePath(applicationYAMLKey string) {
	if cfg.WintrServerAuth.Credentials.FilePath == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrServerAuth.Credentials.FilePath = os.Getenv(fmt.Sprintf("%s_AUTH_CREDENTIALS_FILE_PATH", module))
		if cfg.WintrServerAuth.Credentials.FilePath == "" {
			cfg.WintrServerAuth.Credentials.FilePath = os.Getenv("AUTH_CREDENTIALS_FILE_PATH")
		}
		if cfg.WintrServerAuth.Credentials.FilePath == "" {
			cfg.WintrServerAuth.Credentials.FilePath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if strings.HasPrefix(strings.TrimSpace(cfg.WintrServerAuth.Credentials.FilePath), "{") {
				cfg.WintrServerAuth.Credentials.FilePath = ""
			}
		}
	}
}
