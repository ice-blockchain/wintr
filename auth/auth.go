// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	firebaseauth "github.com/ice-blockchain/wintr/auth/internal/firebase"
	iceauth "github.com/ice-blockchain/wintr/auth/internal/ice"
	"github.com/ice-blockchain/wintr/time"
)

func New(ctx context.Context, applicationYAMLKey string) Client {
	return &auth{
		fb:  firebaseauth.New(ctx, applicationYAMLKey),
		ice: iceauth.New(applicationYAMLKey),
	}
}

func (a *auth) VerifyToken(ctx context.Context, token string) (*Token, error) {
	var authToken *Token
	if err := iceauth.DetectIceToken(token); err != nil {
		if a.fb == nil {
			return nil, errors.Errorf("non-ice token, but firebase auth is disabled")
		}
		authToken, err = a.fb.VerifyToken(ctx, token)

		return authToken, errors.Wrapf(err, "can't verify fb token:%v", token)
	}
	authToken, err := a.ice.VerifyToken(token)

	return authToken, errors.Wrapf(err, "can't verify ice token:%v", token)
}

func (a *auth) ModifyTokenWithMetadata(token *Token, metadataStr string) (*Token, error) {
	if metadataStr == "" {
		return token, nil
	}
	var metadata jwt.MapClaims
	if err := a.ice.VerifyTokenFields(metadataStr, &metadata); err != nil {
		return nil, errors.Wrapf(err, "invalid metadata token:%v", token)
	}
	if metadata["iss"] != internal.MetadataIssuer {
		return nil, errors.Wrapf(ErrWrongTypeToken, "non-metadata token: %v", metadata["iss"])
	}
	if err := a.checkMetadataOwnership(token.UserID, metadata); err != nil {
		return nil, errors.Wrapf(ErrWrongTypeToken, "token %v does not own metadata %#v", token.UserID, metadata)
	}
	if userID := a.firstRegisteredUserID(metadata); userID != "" {
		token.UserID = userID
	}

	return token, nil
}

func (*auth) checkMetadataOwnership(userID string, metadata jwt.MapClaims) error {
	subMatch := metadata["sub"] != "" && userID == metadata["sub"]
	fbMatch := metadata[FirebaseIDClaim] != "" && userID == metadata[FirebaseIDClaim]
	iceMatch := metadata[IceIDClaim] != "" && userID == metadata[IceIDClaim]
	if userID != "" && !(subMatch || fbMatch || iceMatch) {
		return errors.Wrapf(ErrWrongTypeToken, "token %v does not own metadata %#v", userID, metadata)
	}

	return nil
}

func (*auth) firstRegisteredUserID(metadata map[string]any) string {
	var userID string
	if registeredWithProviderInterface, found := metadata[internal.RegisteredWithProviderClaim]; found {
		registeredWithProvider := registeredWithProviderInterface.(string) //nolint:errcheck,revive,forcetypeassert // Not needed.
		switch registeredWithProvider {
		case internal.ProviderFirebase:
			if firebaseIDInterface, ok := metadata[internal.FirebaseIDClaim]; ok {
				userID, _ = firebaseIDInterface.(string) //nolint:errcheck,revive // Not needed.
			}
		case internal.ProviderIce:
			if iceIDInterface, ok := metadata[internal.IceIDClaim]; ok {
				userID, _ = iceIDInterface.(string) //nolint:errcheck,revive // Not needed.
			}
		}
	}

	return userID
}

func (a *auth) UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error {
	if a.fb == nil {
		return nil
	}

	return errors.Wrapf(a.fb.UpdateCustomClaims(ctx, userID, customClaims), "failed to update custom claims for user:%v using firebase auth", userID)
}

func (a *auth) DeleteUser(ctx context.Context, userID string) error {
	if a.fb == nil {
		return nil
	}

	return errors.Wrapf(a.fb.DeleteUser(ctx, userID), "failed to delete user:%v using firebase auth", userID)
}

func (a *auth) UpdateEmail(ctx context.Context, userID, email string) error {
	if a.fb == nil {
		return nil
	}

	return errors.Wrapf(a.fb.UpdateEmail(ctx, userID, email), "failed to update email for user:%v to %v using firebase auth", userID, email)
}

func (a *auth) GetUserUIDByEmail(ctx context.Context, email string) (string, error) {
	if a.fb == nil {
		return "", nil
	}
	usr, err := a.fb.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil
		}

		return "", errors.Wrapf(err, "failed to get user by email:%v using firebase auth", email)
	}

	return usr.UID, nil
}

func (a *auth) GenerateTokens( //nolint:revive // We need to have these parameters.
	now *time.Time, userID, deviceUniqueID, email string, hashCode, seq int64, role string, extras ...map[string]any,
) (accessToken, refreshToken string, err error) {
	var extra map[string]any
	if len(extras) > 0 {
		extra = extras[0]
	}
	accessToken, refreshToken, err = a.ice.GenerateTokens(now, userID, deviceUniqueID, email, hashCode, seq, role, extra)
	err = errors.Wrapf(err, "can't generate tokens for userID:%v, email:%v", userID, email)

	return
}

func (a *auth) GenerateMetadata(
	now *time.Time, tokenID string, metadata map[string]any,
) (string, error) {
	md, err := a.ice.GenerateMetadata(now, tokenID, metadata)

	return md, errors.Wrapf(err, "failed to generate metadata token for tokenID:%v", tokenID)
}

func (a *auth) ParseToken(token string) (*IceToken, error) {
	res := new(IceToken)
	err := a.ice.VerifyTokenFields(token, res)

	return res, errors.Wrapf(err, "can't verify token fields for:%v", token)
}
