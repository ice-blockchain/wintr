// SPDX-License-Identifier: ice License 1.0

package translations

import (
	"context"
	"sync"
	stdlibtime "time"

	"github.com/ice-blockchain/wintr/time"
)

// Public API.

type (
	Language         = string
	TranslationKey   = string
	TranslationValue = string
	TranslationArgs  = map[string]string
	Client           interface {
		Translate(ctx context.Context, lang Language, key TranslationKey, args ...TranslationArgs) (TranslationValue, error)
		TranslateMultipleKeys(ctx context.Context, lang Language, keys []TranslationKey, args ...TranslationArgs) (map[TranslationKey]TranslationValue, error)
		TranslateAllLanguages(ctx context.Context, key TranslationKey, args ...TranslationArgs) (map[Language]TranslationValue, error)
		TranslateMultipleKeysAllLanguages(ctx context.Context, keys []TranslationKey, args ...TranslationArgs) (map[Language]map[TranslationKey]TranslationValue, error) //nolint:lll // .
	}
)

// Private API.

const (
	defaultLanguage = "en"
	refreshInterval = 5 * stdlibtime.Minute
	requestDeadline = 30 * stdlibtime.Second
)

type (
	translations struct {
		cfg                *config
		mx                 *sync.RWMutex
		lastRefreshAt      *time.Time
		data               map[Language]map[TranslationKey]TranslationValue
		defaultLanguage    map[TranslationKey]TranslationValue
		applicationYAMLKey string
	}

	config struct {
		WintrTranslations struct {
			Credentials struct {
				APIKey string `yaml:"apiKey" mapstructure:"apiKey"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/translations" mapstructure:"wintr/translations"` //nolint:tagliatelle // Nope.
	}
)
