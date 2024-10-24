// SPDX-License-Identifier: ice License 1.0

package iceauth

import (
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	appcfg "github.com/ice-blockchain/wintr/config"
)

func New(applicationYAMLKey string) Client {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	cfg.loadSecretForJWT(applicationYAMLKey)

	return &auth{
		cfg: &cfg,
		signToken: func(token *jwt.Token) (string, error) {
			return token.SignedString([]byte(cfg.WintrAuthIce.JWTSecret))
		},
	}
}

func (a *auth) VerifyToken(token string) (*internal.Token, error) {
	var iceToken Token
	err := a.VerifyTokenFields(token, &iceToken)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid token")
	}
	if iceToken.Issuer != internal.AccessJwtIssuer {
		return nil, errors.Wrapf(ErrWrongTypeToken, "access to endpoint with refresh token: %v", iceToken.Issuer)
	}
	tok := &internal.Token{
		Claims: map[string]any{
			"email":          iceToken.Email,
			"role":           iceToken.Role,
			"seq":            iceToken.Seq,
			"hashCode":       iceToken.HashCode,
			"deviceUniqueID": iceToken.DeviceUniqueID,
		},
		UserID:   iceToken.Subject,
		Email:    iceToken.Email,
		Role:     iceToken.Role,
		Provider: internal.ProviderIce,
	}
	if len(iceToken.Claims) > 0 {
		for k, v := range iceToken.Claims {
			if _, alreadyPresented := tok.Claims[k]; !alreadyPresented {
				tok.Claims[k] = v
			}
		}
	}

	return tok, nil
}

func (a *auth) VerifyTokenFields(jwtToken string, res jwt.Claims) error {
	if _, err := jwt.ParseWithClaims(jwtToken, res, a.verify()); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return errors.Wrapf(ErrExpiredToken, "expired or not valid yet token")
		}

		return errors.Wrapf(err, "invalid token:%v", jwtToken)
	}

	return nil
}

func DetectIceToken(jwtToken string) (*Token, error) {
	parser := jwt.NewParser()
	var claims Token
	token, _, err := parser.ParseUnverified(jwtToken, &claims)
	if err != nil {
		return nil, errors.Wrapf(err, "parse unverified error for token:%v", jwtToken)
	}
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
		return nil, errors.Errorf("unexpected signing method:%v", token.Header["alg"])
	}
	if iss, iErr := token.Claims.GetIssuer(); iErr != nil || (iss != internal.AccessJwtIssuer && iss != internal.RefreshJwtIssuer) {
		return nil, errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
	}

	return &claims, nil
}

func (a *auth) verify() func(token *jwt.Token) (any, error) {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		iss, err := token.Claims.GetIssuer()
		invalidIssuer := (iss != internal.AccessJwtIssuer && iss != internal.RefreshJwtIssuer && iss != internal.MetadataIssuer)
		if err != nil || invalidIssuer {
			return nil, errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
		}

		return []byte(a.cfg.WintrAuthIce.JWTSecret), nil
	}
}

func (cfg *config) loadSecretForJWT(applicationYAMLKey string) {
	if cfg.WintrAuthIce.JWTSecret == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrAuthIce.JWTSecret = os.Getenv(module + "_JWT_SECRET")
		if cfg.WintrAuthIce.JWTSecret == "" {
			cfg.WintrAuthIce.JWTSecret = os.Getenv("JWT_SECRET")
		}
	}
}
