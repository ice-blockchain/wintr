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
		Translate(context.Context, Language, TranslationKey, ...TranslationArgs) (TranslationValue, error)
		TranslateMultipleKeys(context.Context, Language, []TranslationKey, ...TranslationArgs) (map[TranslationKey]TranslationValue, error)
		TranslateAllLanguages(context.Context, TranslationKey, ...TranslationArgs) (map[Language]TranslationValue, error)
		TranslateMultipleKeysAllLanguages(context.Context, []TranslationKey, ...TranslationArgs) (map[Language]map[TranslationKey]TranslationValue, error)
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
