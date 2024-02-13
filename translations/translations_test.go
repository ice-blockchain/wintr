// SPDX-License-Identifier: ice License 1.0

package translations

import (
	"context"
	_ "embed"
	"strings"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed .testdata/expected_en
	expectedENTranslation string
	//go:embed .testdata/expected_ru
	expectedRUTranslation string
)

func recoverUnconfigured(t *testing.T) {
	t.Helper()
	if e := recover(); e != nil { //nolint:revive // .
		if err := e.(error); strings.Contains(err.Error(), "Missing API key") { //nolint:errcheck,forcetypeassert,revive // .
			t.Skip("Missing API key")
		}
		if err := e.(error); strings.Contains(err.Error(), "Invalid project key") { //nolint:errcheck,forcetypeassert,revive // .
			t.Skip("Invalid project key")
		}
	}
}

func TestClientTranslate(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	defer func() { recoverUnconfigured(t) }()
	cl := New(ctx, "self")

	translationRU, err := cl.Translate(ctx, "ru", "test", map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	assert.Equal(t, expectedRUTranslation, translationRU)

	translationBogusLanguage, err := cl.Translate(ctx, "zz", "test", map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	assert.Equal(t, expectedENTranslation, translationBogusLanguage)

	translationEN, err := cl.Translate(ctx, "en", "test", map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	assert.Equal(t, translationEN, translationBogusLanguage)
}

func TestClientTranslateAllLanguages(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	defer func() { recoverUnconfigured(t) }()
	cl := New(ctx, "self")

	allTranslations, err := cl.TranslateAllLanguages(ctx, "test", map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(allTranslations), 2)
	assert.Equal(t, expectedRUTranslation, allTranslations["ru"])
	assert.Equal(t, expectedENTranslation, allTranslations["en"])
}

func TestClientTranslateMultipleKeysAllLanguages(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	defer func() { recoverUnconfigured(t) }()
	cl := New(ctx, "self")

	args := map[string]string{"username": "@jdoe", "bogus": "bogus"}
	allTranslations, err := cl.TranslateMultipleKeysAllLanguages(ctx, []TranslationKey{"test", "test"}, args)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(allTranslations), 2)
	require.Len(t, allTranslations["en"], 1)
	require.Len(t, allTranslations["ru"], 1)
	assert.Equal(t, expectedRUTranslation, allTranslations["ru"]["test"])
	assert.Equal(t, expectedENTranslation, allTranslations["en"]["test"])
}

func TestClientTranslateMultipleKeys(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	defer func() { recoverUnconfigured(t) }()
	cl := New(ctx, "self")

	keys := []TranslationKey{"test", "test"}
	translationRU, err := cl.TranslateMultipleKeys(ctx, "ru", keys, map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	require.Len(t, translationRU, 1)
	assert.Equal(t, expectedRUTranslation, translationRU["test"])

	translationBogusLanguage, err := cl.TranslateMultipleKeys(ctx, "zz", keys, map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	require.Len(t, translationBogusLanguage, 1)
	assert.Equal(t, expectedENTranslation, translationBogusLanguage["test"])

	translationEN, err := cl.TranslateMultipleKeys(ctx, "en", keys, map[string]string{"username": "@jdoe", "bogus": "bogus"})
	require.NoError(t, err)
	assert.Equal(t, translationEN, translationBogusLanguage)
}
