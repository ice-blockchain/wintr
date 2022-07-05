package multimedia

import (
	"context"
	"mime/multipart"
)

// Public API.

type (
	Client interface {
		UploadPicture(context.Context, *multipart.FileHeader) error
	}
)

// Private API.

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	multimedia struct{}

	config struct {
		PictureStorage struct {
			URLUpload   string `yaml:"urlUpload"`
			URLDownload string `yaml:"urlDownload"`
			AccessKey   string `yaml:"accessKey"`
		} `yaml:"pictureStorage"`
	}
)
