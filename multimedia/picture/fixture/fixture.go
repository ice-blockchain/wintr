// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	appCfg "github.com/ice-blockchain/wintr/config"
)

func MultipartFileHeader(tb testing.TB, fs embed.FS, filename, multiPartKey, multiPartFileName string) *multipart.FileHeader {
	tb.Helper()
	pic, fErr := fs.Open(fmt.Sprintf(".testdata/%v", filename))
	require.NoError(tb, fErr)
	defer require.NoError(tb, pic.Close())
	stat, err := pic.Stat()
	require.NoError(tb, err)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	formFile, err := writer.CreateFormFile(multiPartKey, multiPartFileName)
	require.NoError(tb, err)
	_, err = io.Copy(formFile, pic)
	require.NoError(tb, err)
	require.NoError(tb, writer.Close())
	form, err := multipart.NewReader(body, writer.Boundary()).ReadForm(stat.Size())
	require.NoError(tb, err)
	require.Greater(tb, len(form.File[multiPartKey]), 0)

	return form.File[multiPartKey][0]
}

func AssertPictureUploaded(ctx context.Context, tb testing.TB, applicationYAMLKey, fileName string) { //nolint:funlen // .
	tb.Helper()
	var cfg struct {
		WintrMultimediaPicture struct {
			Credentials struct {
				AccessKey string `yaml:"accessKey"`
			} `yaml:"credentials" mapstructure:"credentials"`
			URLUpload   string `yaml:"urlUpload"`
			URLDownload string `yaml:"urlDownload"`
		} `yaml:"wintr/multimedia/picture" mapstructure:"wintr/multimedia/picture"` //nolint:tagliatelle // Nope.
	}
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrMultimediaPicture.Credentials.AccessKey == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrMultimediaPicture.Credentials.AccessKey = os.Getenv(fmt.Sprintf("%s_PICTURE_STORAGE_ACCESS_KEY", module))
		if cfg.WintrMultimediaPicture.Credentials.AccessKey == "" {
			cfg.WintrMultimediaPicture.Credentials.AccessKey = os.Getenv("PICTURE_STORAGE_ACCESS_KEY")
		}
	}
	url := fmt.Sprintf("%s/%s", cfg.WintrMultimediaPicture.URLUpload, fileName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	require.NoError(tb, err)
	req.Header.Set("AccessKey", cfg.WintrMultimediaPicture.Credentials.AccessKey)
	//nolint:gosec // Skip checking cert chain from CDN
	httpClient := &http.Client{Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := httpClient.Do(req)
	defer func() {
		require.NoError(tb, resp.Body.Close())
	}()
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusOK, resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(tb, err)
	assert.Greater(tb, len(bodyBytes), 0)

	url = fmt.Sprintf("%s/%s", cfg.WintrMultimediaPicture.URLDownload, fileName)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	require.NoError(tb, err)
	resp, err = httpClient.Do(req)
	defer func() {
		require.NoError(tb, resp.Body.Close())
	}()
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusOK, resp.StatusCode)
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(tb, err)
	assert.Greater(tb, len(bodyBytes), 0)
}

func AssertPictureDeleted(ctx context.Context, tb testing.TB, applicationYAMLKey, fileName string) { //nolint:funlen // .
	tb.Helper()
	var cfg struct {
		WintrMultimediaPicture struct {
			Credentials struct {
				AccessKey string `yaml:"accessKey"`
			} `yaml:"credentials" mapstructure:"credentials"`
			URLUpload   string `yaml:"urlUpload"`
			URLDownload string `yaml:"urlDownload"`
		} `yaml:"wintr/multimedia/picture" mapstructure:"wintr/multimedia/picture"` //nolint:tagliatelle // Nope.
	}
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrMultimediaPicture.Credentials.AccessKey == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrMultimediaPicture.Credentials.AccessKey = os.Getenv(fmt.Sprintf("%s_PICTURE_STORAGE_ACCESS_KEY", module))
		if cfg.WintrMultimediaPicture.Credentials.AccessKey == "" {
			cfg.WintrMultimediaPicture.Credentials.AccessKey = os.Getenv("PICTURE_STORAGE_ACCESS_KEY")
		}
	}
	url := fmt.Sprintf("%s/%s", cfg.WintrMultimediaPicture.URLUpload, fileName)
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	require.NoError(tb, err)
	r.Header.Set("AccessKey", cfg.WintrMultimediaPicture.Credentials.AccessKey)
	//nolint:gosec // Skip checking cert chain from CDN
	httpClient := &http.Client{Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := httpClient.Do(r)
	defer func() {
		require.NoError(tb, resp.Body.Close())
	}()
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusNotFound, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	require.NoError(tb, err)
	assert.True(tb, strings.Contains(string(b), "Object Not Found"))
}
