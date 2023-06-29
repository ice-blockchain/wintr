// SPDX-License-Identifier: ice License 1.0

package iceauth

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	appCfg "github.com/ice-blockchain/wintr/config"
)

func New(applicationYAMLKey string) Client {
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	cfg.loadSecretForJWT(applicationYAMLKey)

	return &auth{
		cfg: &cfg,
		signToken: func(token *jwt.Token) (string, error) {
			return token.SignedString([]byte(cfg.WintrAuthIce.JWTSecret)) //nolint:wrapcheck // .
		},
	}
}

//nolint:funlen // Claims.
func (a *auth) VerifyToken(token string) (*internal.Token, error) {
	var iceToken Token
	err := a.VerifyTokenFields(token, &iceToken)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid email token:%v", token)
	}
	if iceToken.Issuer != internal.AccessJwtIssuer {
		return nil, errors.Wrapf(ErrWrongTypeToken, "access to endpoint with refresh token: %v", iceToken.Issuer)
	}
	userID := iceToken.Subject
	if iceToken.Custom != nil { //nolint:nestif // .
		claims := *iceToken.Custom
		if registeredWithProviderInterface, found := claims[internal.RegisteredWithProviderClaim]; found {
			registeredWithProvider := registeredWithProviderInterface.(string) //nolint:errcheck,forcetypeassert // Not needed.
			if registeredWithProvider == internal.ProviderFirebase {
				if firebaseIDInterface, ok := claims[FirebaseIDClaim]; ok {
					userID, _ = firebaseIDInterface.(string) //nolint:errcheck // Not needed.
				}
			}
		}
	}

	tok := &internal.Token{
		Claims: map[string]any{
			"email":          iceToken.Email,
			"role":           iceToken.Role,
			"seq":            iceToken.Seq,
			"hashCode":       iceToken.HashCode,
			"deviceUniqueID": iceToken.DeviceUniqueID,
		},
		UserID:   userID,
		Email:    iceToken.Email,
		Role:     iceToken.Role,
		Provider: internal.ProviderIce,
	}
	if iceToken.Custom != nil {
		for claimKey, claimValue := range *iceToken.Custom {
			tok.Claims[claimKey] = claimValue
		}
	}

	return tok, nil
}

func (a *auth) VerifyTokenFields(jwtToken string, res jwt.Claims) error {
	if _, err := jwt.ParseWithClaims(jwtToken, res, a.verify()); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return errors.Wrapf(ErrExpiredToken, "expired or not valid yet token:%v", jwtToken)
		}

		return errors.Wrapf(err, "invalid token:%v", jwtToken)
	}

	return nil
}

func DetectIceToken(jwtToken string) error {
	parser := jwt.NewParser()
	var claims Token
	token, _, err := parser.ParseUnverified(jwtToken, &claims)
	if err != nil {
		return errors.Wrapf(err, "parse unverified error for token:%v", jwtToken)
	}
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
		return errors.Errorf("unexpected signing method:%v", token.Header["alg"])
	}
	if iss, iErr := token.Claims.GetIssuer(); iErr != nil || (iss != internal.AccessJwtIssuer && iss != internal.RefreshJwtIssuer) {
		return errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
	}

	return nil
}

func (a *auth) verify() func(token *jwt.Token) (any, error) {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if iss, err := token.Claims.GetIssuer(); err != nil || (iss != internal.AccessJwtIssuer && iss != internal.RefreshJwtIssuer) {
			return nil, errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
		}

		return []byte(a.cfg.WintrAuthIce.JWTSecret), nil
	}
}

func (cfg *config) loadSecretForJWT(applicationYAMLKey string) {
	if cfg.WintrAuthIce.JWTSecret == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrAuthIce.JWTSecret = os.Getenv(fmt.Sprintf("%s_JWT_SECRET", module))
		if cfg.WintrAuthIce.JWTSecret == "" {
			cfg.WintrAuthIce.JWTSecret = os.Getenv("JWT_SECRET")
		}
	}
}
