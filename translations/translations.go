// SPDX-License-Identifier: ice License 1.0

package translations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	stdlibtime "time"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func New(ctx context.Context, applicationYAMLKey string) Client {
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrTranslations.Credentials.APIKey == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrTranslations.Credentials.APIKey = os.Getenv(fmt.Sprintf("%s_TRANSLATIONS_CLIENT_APIKEY", module))
		if cfg.WintrTranslations.Credentials.APIKey == "" {
			cfg.WintrTranslations.Credentials.APIKey = os.Getenv("TRANSLATIONS_CLIENT_APIKEY")
		}
	}

	tc := &translations{
		applicationYAMLKey: applicationYAMLKey,
		cfg:                &cfg,
		mx:                 new(sync.RWMutex),
	}
	log.Panic(tc.downloadAndSetLatestTranslations(ctx)) //nolint:revive // It's by design.
	go tc.startRefreshTranslationsProcess(ctx)

	return tc
}

func (t *translations) Translate(ctx context.Context, language Language, key TranslationKey, args ...TranslationArgs) (TranslationValue, error) {
	if ctx.Err() != nil {
		return "", errors.Wrap(ctx.Err(), "context error")
	}
	t.mx.RLock()
	defer t.mx.RUnlock()

	return t.translate(language, key, args...)
}

func (t *translations) TranslateMultipleKeys(
	ctx context.Context, language Language, keys []TranslationKey, args ...TranslationArgs,
) (map[TranslationKey]TranslationValue, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "context error")
	}
	t.mx.RLock()
	defer t.mx.RUnlock()

	kv := make(map[TranslationKey]TranslationValue, len(keys))
	for _, key := range keys {
		translatedValue, err := t.translate(language, key, args...)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to translate for language %v and key %v", language, key)
		}
		kv[key] = translatedValue
	}

	return kv, nil
}

func (t *translations) translate(language Language, key TranslationKey, args ...TranslationArgs) (TranslationValue, error) { //nolint:revive // Its internal.
	allLanguageTranslations, found := t.data[language]
	if !found {
		if allLanguageTranslations = t.defaultLanguage; allLanguageTranslations == nil {
			return "", errors.Errorf(`neither %q or %q locales have been found`, language, defaultLanguage)
		}
	}
	translated, found := allLanguageTranslations[key]
	if !found {
		if t.defaultLanguage == nil {
			return "", errors.Errorf(`neither %q or %q locales have key %q`, language, defaultLanguage, key)
		}
		translated, found = t.defaultLanguage[key]
		if !found {
			return "", errors.Errorf(`neither %q or %q locales have key %q`, language, defaultLanguage, key)
		}
	}
	if len(args) == 1 {
		for k, v := range args[0] {
			translated = strings.ReplaceAll(translated, fmt.Sprintf("{{%s}}", k), v)
		}
	}

	return translated, nil
}

func (t *translations) TranslateAllLanguages(ctx context.Context, key TranslationKey, args ...TranslationArgs) (map[Language]TranslationValue, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "context error")
	}
	t.mx.RLock()
	defer t.mx.RUnlock()

	allTranslations := make(map[Language]TranslationValue, len(t.data))
	for language := range t.data {
		translatedValue, err := t.translate(language, key, args...)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to translate for language %v and key %v", language, key)
		}
		allTranslations[language] = translatedValue
	}

	return allTranslations, nil
}

func (t *translations) TranslateMultipleKeysAllLanguages(
	ctx context.Context, keys []TranslationKey, args ...TranslationArgs,
) (map[Language]map[TranslationKey]TranslationValue, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "context error")
	}
	t.mx.RLock()
	defer t.mx.RUnlock()

	allTranslations := make(map[Language]map[TranslationKey]TranslationValue, len(t.data))
	for language := range t.data {
		kv := make(map[TranslationKey]TranslationValue, len(keys))
		for _, key := range keys {
			translatedValue, err := t.translate(language, key, args...)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to translate for language %v and key %v", language, key)
			}
			kv[key] = translatedValue
		}
		allTranslations[language] = kv
	}

	return allTranslations, nil
}

func (t *translations) startRefreshTranslationsProcess(ctx context.Context) {
	ticker := stdlibtime.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Debug("started refreshing translations")
			log.Error(errors.Wrap(t.downloadAndSetLatestTranslations(ctx), "failed to downloadAndSetLatestTranslations"),
				"elapsedMinutesSinceLatestRefresh", stdlibtime.Since(*t.lastRefreshAt.Time).Minutes())
			log.Debug("finished refreshing translations")
		}
	}
}

func (t *translations) downloadAndSetLatestTranslations(ctx context.Context) (err error) { //nolint:funlen // Better like this.
	reqCtx, cancel := context.WithTimeout(ctx, requestDeadline)
	defer cancel()
	url := "https://localise.biz/api/export/all.json?index=id&format=multi&status=translated&fallback=en&order=id&key=%s"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fmt.Sprintf(url, t.cfg.WintrTranslations.Credentials.APIKey), http.NoBody)
	if err != nil {
		return errors.Wrapf(err, "failed to build get latest translations request for %q", t.applicationYAMLKey)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to get latest translations for %q", t.applicationYAMLKey)
	}
	defer func() {
		err = multierror.Append(err, errors.Wrap(res.Body.Close(), "failed to close `trying to get latest translations` request body")).ErrorOrNil()
	}()
	if res.StatusCode != http.StatusOK {
		bodyBytes, rErr := io.ReadAll(res.Body)

		return errors.Wrapf(multierror.Append(
			errors.Wrapf(rErr, "failed to read response body of failed `trying to get latest translations` request for %q", t.applicationYAMLKey),
			errors.Errorf("unexpected status code %v while trying to get latest translations for %q, response: %v",
				res.StatusCode, t.applicationYAMLKey, string(bodyBytes))).ErrorOrNil(), "failed to get latest translations")
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read response body of successful `trying to get latest translations` request for %q", t.applicationYAMLKey)
	}
	data := make(map[Language]map[TranslationKey]TranslationValue)
	err = json.UnmarshalContext(ctx, bodyBytes, &data)
	if err != nil {
		return errors.Wrapf(err, "failed to json.unmarshal translated data for %q, into %T, data: %v", t.applicationYAMLKey, data, string(bodyBytes))
	}
	t.mx.Lock()
	t.data = data
	t.defaultLanguage = t.data[defaultLanguage]
	t.lastRefreshAt = time.Now()
	t.mx.Unlock()

	return nil
}
