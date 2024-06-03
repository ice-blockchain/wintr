// SPDX-License-Identifier: ice License 1.0

package picture

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	stdlibtime "time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	"github.com/pkg/errors"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func init() { //nolint:gochecknoinits // It's the only way to tweak the client.
	req.DefaultClient().SetJsonMarshal(json.Marshal)
	req.DefaultClient().SetJsonUnmarshal(json.Unmarshal)
	req.DefaultClient().GetClient().Timeout = requestDeadline
}

func New(applicationYAMLKey string, ignoreFilesRegex ...string) Client {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.WintrMultimediaPicture.Credentials.AccessKey == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrMultimediaPicture.Credentials.AccessKey = os.Getenv(module + "_PICTURE_STORAGE_ACCESS_KEY")
		if cfg.WintrMultimediaPicture.Credentials.AccessKey == "" {
			cfg.WintrMultimediaPicture.Credentials.AccessKey = os.Getenv("PICTURE_STORAGE_ACCESS_KEY")
		}
	}

	var ignFileRegx *regexp.Regexp
	if len(ignoreFilesRegex) == 1 {
		ignFileRegx = regexp.MustCompile(ignoreFilesRegex[0])
	}

	pictureClient := &picture{cfg: &cfg, ignoreFilesRegex: ignFileRegx}
	if cfg.WintrMultimediaPicture.URLUpload != "" {
		resp, err := pictureClient.pictureReq(context.Background()).Delete(pictureClient.uploadURL(uuid.NewString() + ".jpg"))
		log.Panic(err) //nolint:revive // It's intended.
		if resp.GetStatusCode() != http.StatusNotFound {
			log.Panic(errors.New("bootstrap failed"))
		}
	}

	return pictureClient
}

func (p *picture) uploadURL(filename string) string {
	if filename == "" || strings.HasPrefix(filename, p.cfg.WintrMultimediaPicture.URLUpload) {
		return filename
	}

	return fmt.Sprintf("%s/%s", p.cfg.WintrMultimediaPicture.URLUpload, filename)
}

func (p *picture) DownloadURL(name string) string {
	if name == "" || strings.HasPrefix(name, p.cfg.WintrMultimediaPicture.URLDownload) {
		return name
	}

	return fmt.Sprintf("%s/%s", p.cfg.WintrMultimediaPicture.URLDownload, name)
}

func (p *picture) StripDownloadURL(url string) string {
	if url == "" || !strings.HasPrefix(url, p.cfg.WintrMultimediaPicture.URLDownload) {
		return url
	}

	return strings.Replace(url, p.cfg.WintrMultimediaPicture.URLDownload+"/", "", 1)
}

func (p *picture) SQLAliasDownloadURL(name string) string {
	return fmt.Sprintf("'%s/' || %s", p.cfg.WintrMultimediaPicture.URLDownload, name)
}

func (p *picture) UploadPicture(ctx context.Context, data *multipart.FileHeader, oldFileName string) (err error) {
	defer func() {
		if err == nil {
			err = errors.Wrapf(p.doDeletePicture(ctx, oldFileName), "failed to delete old picture %v", oldFileName)
		}
	}()
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

	return errors.Wrapf(p.doUploadPicture(ctx, data, fileData), "error uploading picture %v", data.Filename)
}

func (p *picture) doUploadPicture(ctx context.Context, data *multipart.FileHeader, fileData []byte) error {
	resp, err := p.pictureReq(ctx).
		SetHeader("Content-Type", data.Header.Get("Content-Type")).
		SetBodyBytes(fileData).
		Put(p.uploadURL(data.Filename))
	if err == nil && resp.IsSuccessState() {
		return nil
	}
	if err == nil && !resp.IsSuccessState() {
		body, rErr := resp.ToString()
		log.Error(rErr)

		err = errors.Errorf("upload new picture failed with status: %v,body: %v", resp.GetStatusCode(), body)
	}

	return errors.Wrap(err, "upload picture request failed")
}

func (p *picture) doDeletePicture(ctx context.Context, name string) error {
	filename := name
	if filename == "" {
		return nil
	}
	downloadURL := p.DownloadURL("*")
	filename = strings.Replace(filename, downloadURL[:len(downloadURL)-1], "", 1)
	if p.ignoreFilesRegex != nil && p.ignoreFilesRegex.MatchString(filename) {
		return nil
	}
	resp, err := p.pictureReq(ctx).Delete(p.uploadURL(filename))
	if err == nil && (resp.IsSuccessState() || resp.GetStatusCode() == 404) {
		return nil
	}
	if err == nil && !resp.IsSuccessState() && resp.GetStatusCode() != 404 {
		body, rErr := resp.ToString()
		log.Error(rErr)

		err = errors.Errorf("delete picture failed with status: %v,body: %v", resp.GetStatusCode(), body)
	}

	return errors.Wrap(err, "delete picture request failed")
}

//nolint:mnd,gomnd // Static config.
func (p *picture) pictureReq(ctx context.Context) *req.Request {
	return req.
		SetContext(ctx).
		SetRetryBackoffInterval(10*stdlibtime.Millisecond, 1*stdlibtime.Second). //nolint:mnd,gomnd // .
		SetRetryHook(func(resp *req.Response, err error) {
			switch {
			case err != nil:
				log.Error(errors.Wrapf(err, "failed to upload picture, retrying... "))
			case resp.GetStatusCode() == http.StatusTooManyRequests:
				log.Error(errors.New("rate limit for upload picture reached, retrying... "))
			case resp.GetStatusCode() >= http.StatusInternalServerError:
				log.Error(errors.New("failed to upload picture[internal server error], retrying... "))
			}
		}).
		SetRetryCount(25).
		SetRetryCondition(func(resp *req.Response, err error) bool {
			return err != nil || resp.GetStatusCode() == http.StatusTooManyRequests || resp.GetStatusCode() >= http.StatusInternalServerError
		}).
		SetHeader("AccessKey", p.cfg.WintrMultimediaPicture.Credentials.AccessKey)
}
