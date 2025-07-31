// SPDX-License-Identifier: ice License 1.0

package tracking

import (
	"context"
	"net/http"
	"os"
	"strings"
	stdlibtime "time"

	"github.com/goccy/go-json"
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

func New(applicationYAMLKey string) Client {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.Tracking.Credentials.AppID == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.Tracking.Credentials.AppID = os.Getenv(module + "_ANALYTICS_TRACKING_APP_ID")
		if cfg.Tracking.Credentials.AppID == "" {
			cfg.Tracking.Credentials.AppID = os.Getenv("ANALYTICS_TRACKING_APP_ID")
		}
	}
	if cfg.Tracking.Credentials.APIKey == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.Tracking.Credentials.APIKey = os.Getenv(module + "_ANALYTICS_TRACKING_API_KEY")
		if cfg.Tracking.Credentials.APIKey == "" {
			cfg.Tracking.Credentials.APIKey = os.Getenv("ANALYTICS_TRACKING_API_KEY")
		}
	}
	cl := &tracking{cfg: &cfg}
	ctx, cancel := context.WithTimeout(context.Background(), requestDeadline)
	defer cancel()
	if err := cl.TrackAction(ctx, "", new(Action)); err == nil || !strings.Contains(err.Error(), "customer_id can not be empty Unicode String") {
		if err == nil {
			err = errors.New("unexpected nil err")
		}
		log.Panic(errors.Wrapf(err, "failed to init analytics/tracking"))
	}

	return cl
}

func (t *tracking) TrackAction(ctx context.Context, userID string, action *Action) error {
	url := t.cfg.Tracking.BaseURL + "/v1/event/" + t.cfg.Tracking.Credentials.AppID
	body := make(map[string]any, 1+1+1+1)
	body["type"] = "event"
	body["customer_id"] = userID
	actionItem := make(map[string]any, 1+1)
	actionItem["action"] = action.Name
	actionItem["attributes"] = action.Attributes
	body["actions"] = append(make([]map[string]any, 0, 1), actionItem)

	return errors.Wrapf(t.post(ctx, url, body), "unable to send post request to `%s`, with body: %#v", url, body)
}

func (t *tracking) SetUserAttributes(ctx context.Context, userID string, attributes map[string]any) error {
	url := t.cfg.Tracking.BaseURL + "/v1/customer/" + t.cfg.Tracking.Credentials.AppID
	body := make(map[string]any, 1+1+1)
	body["type"] = "customer"
	body["customer_id"] = userID
	body["attributes"] = attributes

	return errors.Wrapf(t.post(ctx, url, body), "unable to send post request to `%s`, with body: %#v", url, body)
}

func (t *tracking) DeleteUser(ctx context.Context, userID string) error {
	url := t.cfg.Tracking.BaseURL + "/v1/opengdpr_requests/" + t.cfg.Tracking.Credentials.AppID
	body := make(map[string]any, 1+1+1+1+1)
	body["request_type"] = "erasure"
	body["id"] = userID
	body["submitted_time"] = stdlibtime.Now().Format(stdlibtime.RFC3339)
	identities := make(map[string]string, 1+1)
	identities["identity_type"] = "ID"
	identities["identity_value"] = userID
	body["identities"] = append(make([]map[string]string, 0, 1), identities)
	body["api_version"] = "1.0"

	return errors.Wrapf(t.post(ctx, url, body), "unable to send post request to `%s`, with body: %#v", url, body)
}

func (t *tracking) post(ctx context.Context, url string, body any) error { //nolint:funlen // .
	newReq := t.buildHTTPRequest(ctx)
	if stringBody, ok := body.(string); ok {
		newReq = newReq.SetBodyJsonString(stringBody)
	} else {
		newReq = newReq.SetBodyJsonMarshal(body)
	}
	resp, err := newReq.Post(url)
	if err != nil || resp.IsErrorState() {
		if err == nil {
			respBody, pErr := resp.ToString()
			if pErr != nil {
				err = errors.Wrapf(pErr, "analytics/tracking post `%v` failed, body:%#v, [1]unable to read response body", url, body)
			} else {
				err = errors.Errorf("analytics/tracking post `%v` failed, body:%#v, response: %v", url, body, respBody)
			}
		}

		return errors.Wrapf(err, "analytics/tracking post `%v` failed, body:%#v", url, body)
	}
	respBody, err := resp.ToString()
	if err != nil {
		return errors.Wrapf(err, "analytics/tracking post `%v` failed, body:%#v, [2]unable to read response body", url, body)
	}
	var rep struct {
		Status string `json:"status"`
	}
	if err = json.Unmarshal([]byte(respBody), &rep); err != nil {
		return errors.Wrapf(err, "unmarshalling response for analytics/tracking post `%v` failed, body:%#v, statusCode:%v,response: %v", url, body, resp.GetStatusCode(), respBody) //nolint:lll // .
	}
	if rep.Status != "success" {
		return errors.Errorf("analytics/tracking post `%v` failed, body:%#v, response: %v", url, body, respBody)
	}

	return nil
}

//nolint:mnd,gomnd // Static config.
func (t *tracking) buildHTTPRequest(ctx context.Context) *req.Request {
	return req.
		SetContext(ctx).
		SetRetryBackoffInterval(10*stdlibtime.Millisecond, 1*stdlibtime.Second).
		SetRetryHook(func(resp *req.Response, err error) {
			switch { //nolint:revive // .
			case err != nil:
				log.Error(errors.Wrapf(err, "analytics/tracking request failed, retrying... "))
			case resp.GetStatusCode() == http.StatusTooManyRequests:
				log.Error(errors.New("rate limit for analytics/tracking request reached, retrying... "))
			case resp.GetStatusCode() >= http.StatusInternalServerError:
				log.Error(errors.New("analytics/tracking request failed[internal server error], retrying... "))
			}
		}).
		SetRetryCount(25).
		SetRetryCondition(func(resp *req.Response, err error) bool {
			return err != nil || resp.GetStatusCode() == http.StatusTooManyRequests || resp.GetStatusCode() >= http.StatusInternalServerError
		}).
		SetContentType("application/json").
		SetHeader("Accept", "application/json").
		SetBasicAuth(t.cfg.Tracking.Credentials.AppID, t.cfg.Tracking.Credentials.APIKey)
}
