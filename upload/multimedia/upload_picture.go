package multimedia

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	stdlibtime "time"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYamlKey string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	c := &multimedia{}

	return c
}

func (r *multimedia) UploadPicture(ctx context.Context, data *multipart.FileHeader) error {
	if data == nil || data.Size == 0 {
		return nil
	}
	file, err := data.Open()
	defer func() {
		if err = file.Close(); err != nil {
			log.Error(errors.Wrapf(err, "error closing file %v", data.Filename))
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "error opening file %v", data.Filename)
	}
	fileData, err := io.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "error reading file %v", data.Filename)
	}

	return errors.Wrapf(doUploadProfilePicture(ctx, data, fileData), "error uploading file %v", data.Filename)
}

//nolint:gomnd // Static config.
func doUploadProfilePicture(ctx context.Context, data *multipart.FileHeader, fileData []byte) error {
	_, err := req.
		SetContext(ctx).
		SetRetryBackoffInterval(10*stdlibtime.Millisecond, 1*stdlibtime.Second).
		SetRetryHook(func(resp *req.Response, err error) {
			if err != nil {
				log.Error(errors.Wrapf(err, "failed to uploadProfilePicture, retrying... "))
			} else if resp.StatusCode == http.StatusTooManyRequests {
				log.Error(errors.New("rate limit for uploadProfilePicture reached, retrying... "))
			}
		}).
		SetRetryCount(25).
		SetRetryCondition(func(resp *req.Response, err error) bool {
			return (err != nil) || (resp.StatusCode == http.StatusTooManyRequests)
		}).
		SetHeader("AccessKey", cfg.PictureStorage.AccessKey).
		SetHeader("Content-Type", data.Header.Get("Content-Type")).
		SetBodyBytes(fileData).
		Put(fmt.Sprintf("%s/%s", cfg.PictureStorage.URLUpload, data.Filename))

	return errors.Wrap(err, "upload file request failed")
}
