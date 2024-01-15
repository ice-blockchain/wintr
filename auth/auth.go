// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	firebaseAuth "github.com/ice-blockchain/wintr/auth/internal/firebase"
	iceAuth "github.com/ice-blockchain/wintr/auth/internal/ice"
	"github.com/ice-blockchain/wintr/time"
)

func New(ctx context.Context, applicationYAMLKey string) Client {
	return &auth{
		fb:  firebaseAuth.New(ctx, applicationYAMLKey),
		ice: iceAuth.New(applicationYAMLKey),
	}
}

func (a *auth) VerifyToken(ctx context.Context, token string) (*Token, error) {
	var authToken *Token
	if err := iceAuth.DetectIceToken(token); err != nil {
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
		registeredWithProvider := registeredWithProviderInterface.(string) //nolint:errcheck,forcetypeassert // Not needed.
		switch registeredWithProvider {
		case internal.ProviderFirebase:
			if firebaseIDInterface, ok := metadata[internal.FirebaseIDClaim]; ok {
				userID, _ = firebaseIDInterface.(string) //nolint:errcheck // Not needed.
			}
		case internal.ProviderIce:
			if iceIDInterface, ok := metadata[internal.IceIDClaim]; ok {
				userID, _ = iceIDInterface.(string) //nolint:errcheck // Not needed.
			}
		}
	}

	return userID
}

func (a *auth) UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error {
	return errors.Wrapf(a.fb.UpdateCustomClaims(ctx, userID, customClaims), "failed to update custom claims for user:%v using firebase auth", userID)
}

func (a *auth) DeleteUser(ctx context.Context, userID string) error {
	return errors.Wrapf(a.fb.DeleteUser(ctx, userID), "failed to delete user:%v using firebase auth", userID)
}

func (a *auth) UpdateEmail(ctx context.Context, userID, email string) error {
	return errors.Wrapf(a.fb.UpdateEmail(ctx, userID, email), "failed to update email for user:%v to %v using firebase auth", userID, email)
}

func (a *auth) GetUserUIDByEmail(ctx context.Context, email string) (string, error) {
	usr, err := a.fb.GetUserByEmail(ctx, email)
	if err != nil {
		if err == ErrUserNotFound {
			return "", nil
		}

		return "", errors.Wrapf(err, "failed to get user by email:%v using firebase auth", email)
	}

	return usr.UID, err
}

func (a *auth) GenerateTokens( //nolint:revive // We need to have these parameters.
	now *time.Time, userID, deviceUniqueID, email string, hashCode, seq int64, role string,
) (accessToken, refreshToken string, err error) {
	accessToken, refreshToken, err = a.ice.GenerateTokens(now, userID, deviceUniqueID, email, hashCode, seq, role)
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
