// SPDX-License-Identifier: ice License 1.0

package picture

import (
	"context"
	"mime/multipart"
	"regexp"
	stdlibtime "time"
)

// Public API.

type (
	Client interface {
		SQLAliasDownloadURL(pictureColumnName string) string
		DownloadURL(pictureName string) string
		StripDownloadURL(pictureURL string) string

		UploadPicture(ctx context.Context, newPicture *multipart.FileHeader, oldPictureName string) error
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

type (
	picture struct {
		cfg              *config
		ignoreFilesRegex *regexp.Regexp
	}
	config struct {
		WintrMultimediaPicture struct {
			Credentials struct {
				AccessKey string `yaml:"accessKey"`
			} `yaml:"credentials" mapstructure:"credentials"`
			URLUpload   string `yaml:"urlUpload"`
			URLDownload string `yaml:"urlDownload"`
		} `yaml:"wintr/multimedia/picture" mapstructure:"wintr/multimedia/picture"` //nolint:tagliatelle // Nope.
	}
)
